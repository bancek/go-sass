// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/scss.dart

import (
	goUrl "net/url"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

// ScssParser parses the CSS-compatible SCSS syntax, matching Dart's
// ScssParser in scss.dart.
//
// It configures the StylesheetParser base for brace-delimited children,
// semicolon-separated statements, and both comment styles. The indented
// syntax lives in the Sass parser instead; the plain-CSS parser narrows this
// type further.
type ScssParser struct {
	StylesheetParser
}

// NewScssParser creates a ScssParser over contents, resolving relative URLs
// against url when provided. When parseSelectors is set, style rule
// selectors parse as interpolated selectors rather than raw interpolation.
// Style rule selectors parse as unconstrained values here on purpose: they
// are re-parsed after evaluation, so no selector structure is built yet.
func NewScssParser(contents []byte, url *goUrl.URL, parseSelectors bool) *ScssParser {
	p := &ScssParser{}
	p.Parser = *NewParser(contents, url, nil)
	p.StylesheetParser.parseSelectors = parseSelectors
	p.StylesheetParser.isUseAllowed = true
	p.StylesheetParser.globalVariables = make(map[string]sasscommon.FileSpan)
	p.StylesheetParser.indented = false
	p.StylesheetParser.plainCss = false
	p.StylesheetParser.currentIndentation = func() int { return 0 }
	p.StylesheetParser.styleRuleSelector = func() (*Interpolation, error) {
		val, err := p.almostAnyValue(false)
		if err != nil {
			return nil, err
		}
		return val, nil
	}
	p.StylesheetParser.expectStatementSeparator = func(name string) error {
		// Asserts the scanner sits before a separator (;, }, or EOF).
		// Consumes whitespace but nothing else, including comments.
		if err := p.whitespaceWithoutComments(true); err != nil {
			return err
		}
		if p.scanner.IsDone() {
			return nil
		}
		next := p.scanner.PeekChar(0)
		if next == ';' || next == '}' {
			return nil
		}
		return p.expectChar(';')
	}
	p.StylesheetParser.atEndOfStatement = func() bool {
		next := p.scanner.PeekChar(0)
		return next < 0 || next == ';' || next == '}' || next == '{'
	}
	p.StylesheetParser.lookingAtChildren = func() (bool, error) {
		return p.scanner.PeekChar(0) == '{', nil
	}
	p.StylesheetParser.scanElse = func(ifIndentation int) (bool, error) {
		// Only the rule name is scanned here. The ifIndentation parameter is
		// unused outside the indented syntax. An @elseif match records the
		// deprecation warning and rewinds two characters so the caller
		// re-reads "if" as "@else if".
		start := p.scanner.State()
		if err := p.whitespace(true); err != nil {
			return false, err
		}
		beforeAt := p.scanner.State()
		if p.scanner.ScanChar('@') {
			ok, err := p.scanIdentifier("else", true)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
			ok, err = p.scanIdentifier("elseif", true)
			if err != nil {
				return false, err
			}
			if ok {
				span, err := p.spanFrom(beforeAt)
				if err != nil {
					return false, err
				}
				p.warnings = append(p.warnings, ParseTimeWarning{
					Deprecation: deprecation.Elseif,
					Message: "@elseif is deprecated and will not be supported in future Sass " +
						"versions.\n\nRecommendation: @else if",
					Span: span,
				})
				if err := p.setPosition(p.scanner.Position() - 2); err != nil {
					return false, err
				}
				return true, nil
			}
		}
		p.scanner.SetState(start)
		return false, nil
	}
	p.StylesheetParser.children = func(child func() (Statement, error)) ([]Statement, error) {
		// Unlike most consumers this does not consume trailing whitespace, so
		// the parent rule's span doesn't cover whitespace after the rule.
		// Dollar-led lines parse as variable declarations without a
		// namespace; stray semicolons are skipped.
		if err := p.expectChar('{'); err != nil {
			return nil, err
		}
		if err := p.whitespaceWithoutComments(true); err != nil {
			return nil, err
		}
		var children = make([]Statement, 0)
		for {
			switch ch := p.scanner.PeekChar(0); ch {
			case '$':
				decl, err := p.variableDeclarationWithoutNamespace("", nil)
				if err != nil {
					return nil, err
				}
				children = append(children, decl)
			case '/':
				switch p.scanner.PeekChar(1) {
				case '/':
					comment, err := p.scssSilentComment()
					if err != nil {
						return nil, err
					}
					children = append(children, comment)
					if err := p.whitespaceWithoutComments(true); err != nil {
						return nil, err
					}
				case '*':
					comment, err := p.scssLoudComment()
					if err != nil {
						return nil, err
					}
					children = append(children, comment)
					if err := p.whitespaceWithoutComments(true); err != nil {
						return nil, err
					}
				default:
					stmt, err := child()
					if err != nil {
						return nil, err
					}
					children = append(children, stmt)
				}
			case ';':
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
				if err := p.whitespaceWithoutComments(true); err != nil {
					return nil, err
				}
			case '}':
				if err := p.expectChar('}'); err != nil {
					return nil, err
				}
				return children, nil
			case -1:
				return nil, p.scanner.Error("expected \"}\".", -1, 0)
			default:
				stmt, err := child()
				if err != nil {
					return nil, err
				}
				children = append(children, stmt)
			}
		}
	}
	p.StylesheetParser.statements = func(statement func() (Statement, error)) ([]Statement, error) {
		// Consumes top-level statements until EOF. The statement callback may
		// return nil for input that is consumed but not listed.
		var stmts []Statement
		if err := p.whitespaceWithoutComments(true); err != nil {
			return nil, err
		}
		for !p.scanner.IsDone() {
			switch ch := p.scanner.PeekChar(0); ch {
			case '$':
				decl, err := p.variableDeclarationWithoutNamespace("", nil)
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, decl)
			case '/':
				switch p.scanner.PeekChar(1) {
				case '/':
					comment, err := p.scssSilentComment()
					if err != nil {
						return nil, err
					}
					stmts = append(stmts, comment)
					if err := p.whitespaceWithoutComments(true); err != nil {
						return nil, err
					}
				case '*':
					comment, err := p.scssLoudComment()
					if err != nil {
						return nil, err
					}
					stmts = append(stmts, comment)
					if err := p.whitespaceWithoutComments(true); err != nil {
						return nil, err
					}
				default:
					if st, err := statement(); err != nil {
						return nil, err
					} else if st != nil {
						stmts = append(stmts, st)
					}
				}
			case ';':
				if _, err := p.readChar(); err != nil {
					return nil, err
				}
				if err := p.whitespaceWithoutComments(true); err != nil {
					return nil, err
				}
			case -1:
				return stmts, nil
			default:
				if st, err := statement(); err != nil {
					return nil, err
				} else if st != nil {
					stmts = append(stmts, st)
				}
			}
		}
		return stmts, nil
	}
	return p
}

// scssSilentComment consumes a statement-level silent comment block,
// merging consecutive // lines into one comment (spaces after each newline
// are skipped). It fails in plain CSS. Matches Dart's
// ScssParser._silentComment.
func (p *ScssParser) scssSilentComment() (*SilentComment, error) {
	start := p.scanner.State()
	if err := p.expect("//"); err != nil {
		return nil, err
	}

	for {
		for !p.scanner.IsDone() && !util.IsNewline(p.scanner.PeekChar(0)) {
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
		}
		if p.scanner.IsDone() {
			break
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		if err := p.spaces(); err != nil {
			return nil, err
		}
		if p.scanner.PeekChar(0) != '/' || p.scanner.PeekChar(1) != '/' {
			break
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
		if _, err := p.readChar(); err != nil {
			return nil, err
		}
	}

	span, err := p.spanFrom(start)
	if err != nil {
		return nil, err
	}

	if p.plainCss {
		if err := p.error("Silent comments aren't allowed in plain CSS.", span, nil); err != nil {
			return nil, err
		}
	}

	last := NewSilentComment(p.scanner.Substring(start.Position, nil), span)
	p.lastSilentComment = last
	return last, nil
}

// scssLoudComment consumes a statement-level loud comment block, ending at
// the first */. Carriage returns and form feeds normalize to a newline and
// #{...} becomes interpolation. Matches Dart's ScssParser._loudComment.
func (p *ScssParser) scssLoudComment() (*LoudComment, error) {
	start := p.scanner.State()
	if err := p.expect("/*"); err != nil {
		return nil, err
	}
	buffer := &InterpolationBuffer{}
	buffer.Write("/*")

	for {
		switch ch := p.scanner.PeekChar(0); ch {
		case '#':
			if p.scanner.PeekChar(1) == '{' {
				expr, span, err := p.singleInterpolation()
				if err != nil {
					return nil, err
				}
				buffer.Add(expr, span)
			} else {
				ch, err := p.readChar()
				if err != nil {
					return nil, err
				}
				buffer.WriteCharCode(ch)
			}
		case '*':
			ch, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(ch)
			if p.scanner.PeekChar(0) != '/' {
				continue
			}
			ch, err = p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(ch)
			span, err := p.spanFrom(start)
			if err != nil {
				return nil, err
			}
			comment, commentErr := buffer.Interpolation(span)
			if commentErr != nil {
				return nil, commentErr
			}
			return NewLoudComment(comment), nil
		case '\r':
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			if p.scanner.PeekChar(0) != '\n' {
				buffer.WriteCharCode('\n')
			}
		case '\f':
			if _, err := p.readChar(); err != nil {
				return nil, err
			}
			buffer.WriteCharCode('\n')
		default:
			ch, err := p.readChar()
			if err != nil {
				return nil, err
			}
			buffer.WriteCharCode(ch)
		}
	}
}
