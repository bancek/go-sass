// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/parse/at_root_query.dart

import (
	"strings"
)

// AtRootQueryParser parses `@at-root` queries of the form `(with: ...)` or
// `(without: ...)` followed by a space-separated list of at-rule names.
//
// The parser is constructed by embedding a base Parser (see NewAtRootQuery in
// sass_at_root_query.go); rule names are lowercased so matching is
// case-insensitive.
//
// Matches Dart: AtRootQueryParser in lib/src/parse/at_root_query.dart
type AtRootQueryParser struct {
	Parser
}

// Parse parses an `@at-root` query: `(with|without: <rule>...)` with the
// whole input consumed. `with` sets include mode, `without` sets exclude
// mode; at least one rule name is required and parsing stops at the first
// non-identifier before the closing paren.
//
// Matches Dart: AtRootQueryParser.parse
func (p *AtRootQueryParser) Parse() (*AtRootQuery, error) {
	var result *AtRootQuery
	err := p.wrapSpanFormatException(func() error {
		if err := p.expectChar('('); err != nil {
			return err
		}
		if err := p._whitespace(); err != nil {
			return err
		}
		include, err := p.scanIdentifier("with", false)
		if err != nil {
			return err
		}
		if !include {
			if err := p.expectIdentifier("without", `"with" or "without"`, false); err != nil {
				return err
			}
		}
		if err := p._whitespace(); err != nil {
			return err
		}
		if err := p.expectChar(':'); err != nil {
			return err
		}
		if err := p._whitespace(); err != nil {
			return err
		}

		atRules := make(map[string]struct{})
		for {
			rule, err := p.identifier(false, false)
			if err != nil {
				return err
			}
			atRules[strings.ToLower(rule)] = struct{}{}
			if err := p._whitespace(); err != nil {
				return err
			}
			if !p.lookingAtIdentifier(nil) {
				break
			}
		}
		if err := p.expectChar(')'); err != nil {
			return err
		}
		if err := p.scanner.ExpectDone(); err != nil {
			return err
		}

		result = NewAtRootQuery(atRules, include)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// _whitespace consumes query whitespace. The consume-newlines flag is fixed
// to true here; it is irrelevant for this single-line grammar and simply
// permits newlines anywhere whitespace is allowed.
//
// Matches Dart: AtRootQueryParser._whitespace
func (p *AtRootQueryParser) _whitespace() error {
	return p.whitespace(true)
}
