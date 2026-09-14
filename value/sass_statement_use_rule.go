// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/use_rule.dart

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// UseRule is a @use rule.
//
// It loads another module and binds its public members under a namespace.
type UseRule struct {
	url *url.URL
	// Namespace qualifies members of the used module.
	// Nil means the members are usable without a namespace.
	Namespace *string
	// Configuration holds the variable assignments configuring the module.
	Configuration []*ConfiguredVariable
	span          sasscommon.FileSpan
}

// NewUseRule creates a @use rule loading u under namespace with configuration.
// It reports an error when a configured variable is guarded, which @use
// forbids.
func NewUseRule(u *url.URL, namespace *string, span sasscommon.FileSpan, configuration []*ConfiguredVariable) (*UseRule, error) {
	if configuration == nil {
		configuration = []*ConfiguredVariable{}
	}
	for _, variable := range configuration {
		if variable.IsGuarded {
			return nil, &sasscommon.ArgumentError{Name: "configured variable", Message: fmt.Sprintf("Configured variable %q can't be guarded in a @use rule.", variable.Name())}
		}
	}
	return &UseRule{
		url:           u,
		Namespace:     namespace,
		Configuration: configuration,
		span:          span,
	}, nil
}

// URL returns the URI of the used module.
func (u *UseRule) URL() *url.URL { return u.url }

// URLSpan returns the span covering the quoted module URL.
func (u *UseRule) URLSpan() (sasscommon.FileSpan, error) {
	span, err := u.span.WithoutInitialAtRule()
	if err != nil {
		return nil, err
	}
	return span.InitialQuoted()
}
func (u *UseRule) Span() (sasscommon.FileSpan, error) { return u.span, nil }
func (u *UseRule) IsStatement()                       {}
func (u *UseRule) IsSassNode()                        {}
func (u *UseRule) IsAstNode()                         {}
func (u *UseRule) isSassDependency()                  {}

func (u *UseRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@use ")
	builder.WriteString(QuoteText(u.url.String()))

	basename := urlBasename(u.url)
	if u.Namespace == nil || *u.Namespace != basename {
		builder.WriteString(" as ")
		if u.Namespace == nil {
			builder.WriteString("*")
		} else {
			builder.WriteString(*u.Namespace)
		}
	}

	if len(u.Configuration) > 0 {
		parts := make([]string, len(u.Configuration))
		for i, v := range u.Configuration {
			s, err := v.String()
			if err != nil {
				return "", err
			}
			parts[i] = s
		}
		builder.WriteString(" with (")
		builder.WriteString(strings.Join(parts, ", "))
		builder.WriteString(")")
	}

	builder.WriteString(";")
	return builder.String(), nil
}

func urlBasename(u *url.URL) string {
	path := u.Path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		path = path[idx+1:]
	}
	if dot := strings.Index(path, "."); dot >= 0 {
		path = path[:dot]
	}
	return path
}

// ParseUseRule parses a @use rule from contents.
//
// If passed, url is the name of the file from which contents comes.
// Throws a SassFormatException if parsing fails.
//
// Matches Dart: UseRule.parse
func ParseUseRule(contents string, url *url.URL) (*UseRule, error) {
	p := NewScssParser([]byte(contents), url, false)
	return parseSingleProduction(&p.StylesheetParser, func() (*UseRule, error) {
		start := p.scanner.State()
		if err := p.expectChar('@'); err != nil {
			return nil, err
		}
		if err := p.expectIdentifier("use", "", true); err != nil {
			return nil, err
		}
		if err := p.whitespace(true); err != nil {
			return nil, err
		}
		return p.useRule(start)
	})
}
