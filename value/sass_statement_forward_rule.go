// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/statement/forward_rule.dart

import (
	"net/url"
	"strings"

	"github.com/bancek/go-sass/orderedset"
	"github.com/bancek/go-sass/sasscommon"
)

// ForwardRule is a @forward rule.
//
// It re-exports members of another module, optionally filtered to a shown
// or hidden subset, renamed with a prefix, and reconfigured with variable
// assignments.
type ForwardRule struct {
	url *url.URL
	// ShownMixinsAndFunctions lists the mixin/function names allowed through.
	// Nil means no show restriction; when non-nil, the hidden sets are nil
	// and ShownVariables is non-nil.
	ShownMixinsAndFunctions *orderedset.LinkedSet[string]
	// ShownVariables lists the variable names (without $) allowed through.
	// Nil means no show restriction; see ShownMixinsAndFunctions.
	ShownVariables *orderedset.LinkedSet[string]
	// HiddenMixinsAndFunctions lists the mixin/function names held back.
	// When non-nil, the shown sets are nil and HiddenVariables is non-nil.
	HiddenMixinsAndFunctions *orderedset.LinkedSet[string]
	// HiddenVariables lists the variable names (without $) held back.
	// See HiddenMixinsAndFunctions.
	HiddenVariables *orderedset.LinkedSet[string]
	// Prefix is prepended to forwarded member names, or nil to keep them as-is.
	Prefix *string
	// Configuration holds the variable assignments configuring the module.
	Configuration []*ConfiguredVariable
	span          sasscommon.FileSpan
}

func copyConfiguration(configuration []*ConfiguredVariable) []*ConfiguredVariable {
	if configuration == nil {
		return []*ConfiguredVariable{}
	}
	// Make an immutable copy, matching Dart's List.unmodifiable.
	cfg := make([]*ConfiguredVariable, len(configuration))
	copy(cfg, configuration)
	return cfg
}

// NewForwardRule creates a @forward rule exposing every member of the
// module at u, with an optional name prefix and configuration.
func NewForwardRule(u *url.URL, span sasscommon.FileSpan, prefix *string, configuration []*ConfiguredVariable) *ForwardRule {
	return &ForwardRule{
		url:           u,
		span:          span,
		Prefix:        prefix,
		Configuration: copyConfiguration(configuration),
	}
}

// NewForwardRuleShow creates a @forward rule exposing only the shown mixin,
// function, and variable names.
func NewForwardRuleShow(u *url.URL, shownMixinsAndFunctions, shownVariables *orderedset.LinkedSet[string], span sasscommon.FileSpan, prefix *string, configuration []*ConfiguredVariable) *ForwardRule {
	if shownMixinsAndFunctions == nil {
		shownMixinsAndFunctions = orderedset.New[string]()
	}
	if shownVariables == nil {
		shownVariables = orderedset.New[string]()
	}
	return &ForwardRule{
		url:                     u,
		ShownMixinsAndFunctions: shownMixinsAndFunctions,
		ShownVariables:          shownVariables,
		span:                    span,
		Prefix:                  prefix,
		Configuration:           copyConfiguration(configuration),
	}
}

// NewForwardRuleHide creates a @forward rule exposing every member except
// the hidden mixin, function, and variable names.
func NewForwardRuleHide(u *url.URL, hiddenMixinsAndFunctions, hiddenVariables *orderedset.LinkedSet[string], span sasscommon.FileSpan, prefix *string, configuration []*ConfiguredVariable) *ForwardRule {
	if hiddenMixinsAndFunctions == nil {
		hiddenMixinsAndFunctions = orderedset.New[string]()
	}
	if hiddenVariables == nil {
		hiddenVariables = orderedset.New[string]()
	}
	return &ForwardRule{
		url:                      u,
		HiddenMixinsAndFunctions: hiddenMixinsAndFunctions,
		HiddenVariables:          hiddenVariables,
		span:                     span,
		Prefix:                   prefix,
		Configuration:            copyConfiguration(configuration),
	}
}

// URL returns the URI of the forwarded module.
func (r *ForwardRule) URL() *url.URL { return r.url }

// URLSpan returns the span of the URL in this @forward rule.
//
// Matches Dart: ForwardRule.urlSpan
func (r *ForwardRule) URLSpan() (sasscommon.FileSpan, error) {
	span, err := r.span.WithoutInitialAtRule()
	if err != nil {
		return nil, err
	}
	return span.InitialQuoted()
}
func (r *ForwardRule) Span() (sasscommon.FileSpan, error) { return r.span, nil }
func (r *ForwardRule) IsStatement()                       {}
func (r *ForwardRule) IsSassNode()                        {}
func (r *ForwardRule) IsAstNode()                         {}
func (r *ForwardRule) isSassDependency()                  {}

func (r *ForwardRule) String() (string, error) {
	var builder strings.Builder
	builder.WriteString("@forward ")
	builder.WriteString(QuoteText(r.url.String()))

	shownMixinsAndFunctions := r.ShownMixinsAndFunctions
	hiddenMixinsAndFunctions := r.HiddenMixinsAndFunctions
	if shownMixinsAndFunctions != nil {
		builder.WriteString(" show ")
		builder.WriteString(r.memberList(shownMixinsAndFunctions, r.ShownVariables))
	} else if hiddenMixinsAndFunctions != nil && hiddenMixinsAndFunctions.Len() > 0 {
		builder.WriteString(" hide ")
		builder.WriteString(r.memberList(hiddenMixinsAndFunctions, r.HiddenVariables))
	}

	if r.Prefix != nil {
		builder.WriteString(" as ")
		builder.WriteString(*r.Prefix)
		builder.WriteString("*")
	}

	if len(r.Configuration) > 0 {
		parts := make([]string, len(r.Configuration))
		for i, v := range r.Configuration {
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

// memberList renders the shown/hidden member names, prefixing variables
// with $ and joining everything with commas.
func (r *ForwardRule) memberList(mixinsAndFunctions, variables *orderedset.LinkedSet[string]) string {
	total := 0
	if mixinsAndFunctions != nil {
		total += mixinsAndFunctions.Len()
	}
	if variables != nil {
		total += variables.Len()
	}
	names := make([]string, 0, total)
	if mixinsAndFunctions != nil {
		for name := range mixinsAndFunctions.Keys() {
			names = append(names, name)
		}
	}
	if variables != nil {
		for name := range variables.Keys() {
			names = append(names, "$"+name)
		}
	}
	return strings.Join(names, ", ")
}
