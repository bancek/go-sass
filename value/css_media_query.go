// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/ast/css/media_query.dart

import (
	"errors"
	"net/url"
	"slices"
	"strings"
)

// CssMediaQuery represents a plain CSS media query, as used in @media and @import.
//
// Matches Dart: CssMediaQuery. Modifier is "not"/"only" or nil, Type is the
// media type or nil, Conjunction selects all-vs-any condition matching, and
// Conditions holds parenthesized media-in-parens strings.
type CssMediaQuery struct {
	// Modifier is the query modifier ("not" or "only"), or nil if absent.
	Modifier *string
	// Type is the media type ("screen", "print"), or nil if the query is conditions-only.
	Type *string
	// Conjunction reports whether Conditions must all match (true) or any match (false).
	Conjunction bool
	// Conditions holds media conditions including parentheses.
	Conditions []string
}

// NewCssMediaQueryType creates a media query that specifies a type and,
// optionally, conditions.
//
// Conjunction is always true for this form. Matches Dart:
// CssMediaQuery.type.
func NewCssMediaQueryType(typ *string, modifier *string, conditions []string) *CssMediaQuery {
	conds := make([]string, len(conditions))
	copy(conds, conditions)
	return &CssMediaQuery{
		Type:        typ,
		Modifier:    modifier,
		Conjunction: true,
		Conditions:  conds,
	}
}

// NewCssMediaQueryCondition creates a media query that matches conditions
// according to conjunction.
//
// A nil conjunction defaults to true; more than one condition requires an
// explicit conjunction. Matches Dart: CssMediaQuery.condition.
func NewCssMediaQueryCondition(conditions []string, conjunction *bool) (*CssMediaQuery, error) {
	if conjunction == nil && len(conditions) > 1 {
		return nil, errors.New(
			"If conditions is longer than one element, conjunction may not be null.",
		)
	}
	conds := make([]string, len(conditions))
	copy(conds, conditions)
	c := true
	if conjunction != nil {
		c = *conjunction
	}
	return &CssMediaQuery{
		Conjunction: c,
		Conditions:  conds,
	}, nil
}

// MatchesAllTypes returns whether this media query matches all media types.
//
// True when no type is set or the type is "all" (case-insensitive).
// Matches Dart: CssMediaQuery.matchesAllTypes.
func (q *CssMediaQuery) MatchesAllTypes() bool {
	return q.Type == nil || strings.EqualFold(*q.Type, "all")
}

// Merge merges this with other to return a query that matches the intersection
// of both inputs.
//
// Disjunctive ("or") queries are unrepresentable. Negated queries follow the
// subset rule: `not T and (a)` intersected with `T and (a) and (b)` is empty,
// while disjoint conditions stay unrepresentable. Matches Dart:
// CssMediaQuery.merge.
func (q *CssMediaQuery) Merge(other *CssMediaQuery) MediaQueryMergeResult {
	if !q.Conjunction || !other.Conjunction {
		return MediaQueryMergeResultUnrepresentable
	}

	var ourModifier, ourType, theirModifier, theirType *string
	if q.Modifier != nil {
		v := strings.ToLower(*q.Modifier)
		ourModifier = &v
	}
	if q.Type != nil {
		v := strings.ToLower(*q.Type)
		ourType = &v
	}
	if other.Modifier != nil {
		v := strings.ToLower(*other.Modifier)
		theirModifier = &v
	}
	if other.Type != nil {
		v := strings.ToLower(*other.Type)
		theirType = &v
	}

	if ourType == nil && theirType == nil {
		conditions := make([]string, 0, len(q.Conditions)+len(other.Conditions))
		conditions = append(conditions, q.Conditions...)
		conditions = append(conditions, other.Conditions...)
		return &MediaQuerySuccessfulMergeResult{
			Query: &CssMediaQuery{
				Conjunction: true,
				Conditions:  conditions,
			},
		}
	}

	var modifier *string
	var typ *string
	var conditions []string

	ourNot := ourModifier != nil && *ourModifier == "not"
	theirNot := theirModifier != nil && *theirModifier == "not"

	if ourNot != theirNot {
		ourTypeStr := ""
		if ourType != nil {
			ourTypeStr = *ourType
		}
		theirTypeStr := ""
		if theirType != nil {
			theirTypeStr = *theirType
		}

		if ourTypeStr == theirTypeStr {
			var negativeConditions, positiveConditions []string
			if ourNot {
				negativeConditions = q.Conditions
				positiveConditions = other.Conditions
			} else {
				negativeConditions = other.Conditions
				positiveConditions = q.Conditions
			}

			if allContained(negativeConditions, positiveConditions) {
				return MediaQueryMergeResultEmpty
			}
			return MediaQueryMergeResultUnrepresentable
		} else if q.MatchesAllTypes() || other.MatchesAllTypes() {
			return MediaQueryMergeResultUnrepresentable
		}

		if ourNot {
			modifier = theirModifier
			typ = theirType
			conditions = other.Conditions
		} else {
			modifier = ourModifier
			typ = ourType
			conditions = q.Conditions
		}
	} else if ourNot {
		ourTypeStr := ""
		if ourType != nil {
			ourTypeStr = *ourType
		}
		theirTypeStr := ""
		if theirType != nil {
			theirTypeStr = *theirType
		}

		if ourTypeStr != theirTypeStr {
			return MediaQueryMergeResultUnrepresentable
		}

		var moreConditions, fewerConditions []string
		if len(q.Conditions) > len(other.Conditions) {
			moreConditions = q.Conditions
			fewerConditions = other.Conditions
		} else {
			moreConditions = other.Conditions
			fewerConditions = q.Conditions
		}

		if !allContained(fewerConditions, moreConditions) {
			return MediaQueryMergeResultUnrepresentable
		}

		modifier = ourModifier
		typ = ourType
		conditions = moreConditions
	} else if q.MatchesAllTypes() {
		modifier = theirModifier
		if other.MatchesAllTypes() && ourType == nil {
			typ = nil
		} else {
			typ = theirType
		}
		conditions = append(q.Conditions, other.Conditions...)
	} else if other.MatchesAllTypes() {
		modifier = ourModifier
		typ = ourType
		conditions = append(q.Conditions, other.Conditions...)
	} else {
		ourTypeStr := ""
		if ourType != nil {
			ourTypeStr = *ourType
		}
		theirTypeStr := ""
		if theirType != nil {
			theirTypeStr = *theirType
		}

		if ourTypeStr != theirTypeStr {
			return MediaQueryMergeResultEmpty
		}

		if ourModifier != nil {
			modifier = ourModifier
		} else {
			modifier = theirModifier
		}
		typ = ourType
		conditions = append(q.Conditions, other.Conditions...)
	}

	var resultModifier *string
	if modifier == ourModifier {
		resultModifier = q.Modifier
	} else {
		resultModifier = other.Modifier
	}

	var resultType *string
	if typ == ourType {
		resultType = q.Type
	} else {
		resultType = other.Type
	}

	conds := make([]string, len(conditions))
	copy(conds, conditions)

	return &MediaQuerySuccessfulMergeResult{
		Query: &CssMediaQuery{
			Type:        resultType,
			Modifier:    resultModifier,
			Conjunction: true,
			Conditions:  conds,
		},
	}
}

// allContained reports whether every needle appears in haystack, used for
// the negated-query subset check in Merge.
func allContained(needles, haystack []string) bool {
	for _, n := range needles {
		found := slices.Contains(haystack, n)
		if !found {
			return false
		}
	}
	return true
}

// MediaQueriesEqual returns whether two CssMediaQuery values are equal.
//
// Modifiers, types, conjunctions, and ordered conditions must all match.
// Matches Dart: CssMediaQuery operator==.
func MediaQueriesEqual(a, b *CssMediaQuery) bool {
	if (a.Modifier == nil) != (b.Modifier == nil) {
		return false
	}
	if a.Modifier != nil && *a.Modifier != *b.Modifier {
		return false
	}
	if (a.Type == nil) != (b.Type == nil) {
		return false
	}
	if a.Type != nil && *a.Type != *b.Type {
		return false
	}
	if len(a.Conditions) != len(b.Conditions) {
		return false
	}
	for i := range a.Conditions {
		if a.Conditions[i] != b.Conditions[i] {
			return false
		}
	}
	return true
}

// MediaQueryHashEqual is the equality function for LinkedHashSet[*CssMediaQuery].
func MediaQueryHashEqual(a, b *CssMediaQuery) bool { return MediaQueriesEqual(a, b) }

// HashCode returns a hash code for this media query, combining modifier,
// type, conjunction, and conditions for hash-keyed query sets.
func (q *CssMediaQuery) HashCode() int {
	h := 17
	if q.Modifier != nil {
		h = h*31 + hashString(*q.Modifier)
	}
	if q.Type != nil {
		h = h*31 + hashString(*q.Type)
	}
	if q.Conjunction {
		h = h*31 + 1
	} else {
		h = h*31 + 0
	}
	for _, c := range q.Conditions {
		h = h*31 + hashString(c)
	}
	return h
}

// hashString folds runes with the 31-multiplier used by HashCode.
func hashString(s string) int {
	h := 0
	for _, r := range s {
		h = h*31 + int(r)
	}
	return h
}

// String renders the query as CSS text, joining conditions with "and" or
// "or" per Conjunction.
//
// Matches Dart: CssMediaQuery.toString.
func (q *CssMediaQuery) String() string {
	var b strings.Builder
	if q.Modifier != nil {
		b.WriteString(*q.Modifier)
		b.WriteString(" ")
	}
	if q.Type != nil {
		b.WriteString(*q.Type)
		if len(q.Conditions) > 0 {
			b.WriteString(" and ")
		}
	}
	sep := " and "
	if !q.Conjunction {
		sep = " or "
	}
	for i, c := range q.Conditions {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(c)
	}
	return b.String()
}

// MediaQueryMergeResult is the result of merging two media queries.
//
// Either empty, unrepresentable, or a successful merge carrying the merged
// query. Matches Dart: MediaQueryMergeResult.
type MediaQueryMergeResult interface {
	isMediaQueryMergeResult()
}

type mediaQueryMergeResultEmpty struct{}

func (mediaQueryMergeResultEmpty) isMediaQueryMergeResult() {}

type mediaQueryMergeResultUnrepresentable struct{}

func (mediaQueryMergeResultUnrepresentable) isMediaQueryMergeResult() {}

var (
	// MediaQueryMergeResultEmpty marks a merge whose intersection matches nothing.
	//
	// Matches Dart: MediaQueryMergeResult.empty.
	MediaQueryMergeResultEmpty MediaQueryMergeResult = mediaQueryMergeResultEmpty{}
	// MediaQueryMergeResultUnrepresentable marks a merge CSS cannot express
	// (disjunctions, "neither screen nor print", cross-type negations).
	//
	// Matches Dart: MediaQueryMergeResult.unrepresentable.
	MediaQueryMergeResultUnrepresentable MediaQueryMergeResult = mediaQueryMergeResultUnrepresentable{}
)

// MediaQuerySuccessfulMergeResult is a successful result of merging two media
// queries.
//
// Matches Dart: MediaQuerySuccessfulMergeResult.
type MediaQuerySuccessfulMergeResult struct {
	// Query is the merged query matching the intersection of both inputs.
	Query *CssMediaQuery
}

func (m *MediaQuerySuccessfulMergeResult) isMediaQueryMergeResult() {}

// String returns the merged query's CSS text.
//
// Matches Dart: MediaQuerySuccessfulMergeResult toString.
func (m *MediaQuerySuccessfulMergeResult) String() string {
	return m.Query.String()
}

// ParseList parses a media query from the given contents.
//
// If passed, url is the name of the file from which contents comes.
//
// Matches Dart: CssMediaQuery.parseList
func ParseList(contents string, url *url.URL, interpolationMap *InterpolationMap) ([]*CssMediaQuery, error) {
	p := NewParser([]byte(contents), url, interpolationMap)
	return (&CssMediaQueryParser{Parser: *p}).Parse()
}
