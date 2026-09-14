// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/sass/at_root_query.dart

import (
	"maps"
	"net/url"
	"strings"
)

// AtRootQuery selects which enclosing rules an @at-root block bubbles
// through.
//
// When Include is true the named rules are kept in place and everything else
// is bubbled past; when false the named rules are bubbled past. Two names
// are special: "all" matches every rule and "rule" matches style rules.
type AtRootQuery struct {
	// Include switches the query from exclusion (false) to inclusion (true).
	Include bool
	names   map[string]struct{}
	all     bool
	rule    bool
}

// ParseAtRootQuery parses an at-root query from contents.
//
// Matches Dart: AtRootQuery.parse
func ParseAtRootQuery(contents string, url *url.URL, interpolationMap *InterpolationMap) (*AtRootQuery, error) {
	p := &AtRootQueryParser{Parser: *NewParser([]byte(contents), url, interpolationMap)}
	return p.Parse()
}

// DefaultQuery is the default at-root query, which excludes only style rules.
var DefaultQuery = &AtRootQuery{
	Include: false,
	names:   map[string]struct{}{},
	all:     false,
	rule:    true,
}

// NewAtRootQuery creates a query over names, precomputing the "all" and
// "rule" shortcuts. Include selects inclusion (true) over exclusion (false).
func NewAtRootQuery(names map[string]struct{}, include bool) *AtRootQuery {
	n := make(map[string]struct{}, len(names))
	maps.Copy(n, names)
	_, hasAll := n["all"]
	_, hasRule := n["rule"]
	return &AtRootQuery{
		Include: include,
		names:   n,
		all:     hasAll,
		rule:    hasRule,
	}
}

// ExcludesStyleRules returns whether this excludes style rules, taking Include
// into account.
func (q *AtRootQuery) ExcludesStyleRules() bool {
	return (q.all || q.rule) != q.Include
}

// ExcludesName returns whether this excludes an at-rule with the given name.
func (q *AtRootQuery) ExcludesName(name string) bool {
	_, hasName := q.names[name]
	return (q.all || hasName) != q.Include
}

// Excludes returns whether this excludes node.
func (q *AtRootQuery) Excludes(node CssParentNode) bool {
	if q.all {
		return !q.Include
	}
	switch n := node.(type) {
	case CssStyleRule:
		return q.ExcludesStyleRules()
	case CssMediaRule:
		return q.ExcludesName("media")
	case CssSupportsRule:
		return q.ExcludesName("supports")
	case CssAtRule:
		return q.ExcludesName(strings.ToLower(n.Name().Value))
	}
	return false
}
