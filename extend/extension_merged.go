// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package extend

// dart-source: lib/src/extend/merged_extension.dart
//
import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// MergedExtension folds two extensions with the same extender and target
// into one entry. Merging keeps a single map slot mandatory-aware: both
// branches stay marked resolved downstream, while the merged node itself is
// optional so it does not fire unsatisfied-extension errors on its own. The
// merged span follows the left branch and the media context follows the
// first non-nil branch.
type MergedExtension struct {
	*BaseExtension
	// Left is one of the merged branches.
	Left Extension
	// Right is the other merged branch.
	Right Extension
}

// IsMerged reports true for merged extensions.
func (m *MergedExtension) IsMerged() bool { return true }

// MergeExtensions combines left and right when they share extender and
// target. Mismatched pairs are an argument error; extensions defined in
// different media contexts cannot merge and report the left rule's location
// on the right rule's span. A context-free optional branch adds nothing, so
// the other branch is returned directly; otherwise the branches fold into a
// MergedExtension whose extender points back at the merged node.
func MergeExtensions(left, right Extension) (Extension, error) {
	if !left.Extender().Selector.Equals(right.Extender().Selector) ||
		!left.Target().Equals(right.Target()) {
		leftStr, err := left.String()
		if err != nil {
			return nil, err
		}
		rightStr, err := right.String()
		if err != nil {
			return nil, err
		}
		return nil, &sasscommon.ArgumentError{Message: fmt.Sprintf("%v and %v aren't the same extension.", leftStr, rightStr)}
	}

	if left.MediaContext() != nil && right.MediaContext() != nil &&
		!mediaQueriesEqual(left.MediaContext(), right.MediaContext()) {
		leftSpan, err := left.Span()
		if err != nil {
			return nil, err
		}
		rightSpan, err := right.Span()
		if err != nil {
			return nil, err
		}
		leftURL, err := leftSpan.SourceURL()
		if err != nil {
			return nil, err
		}
		leftStart, err := leftSpan.StartLocation()
		if err != nil {
			return nil, err
		}
		return nil, &sasscommon.SassException{
			Message: fmt.Sprintf("From line %d, column %d of %s: \nYou may not @extend the same selector from within different media queries.",
				leftStart.Line+1, leftStart.Column+1, sasscommon.PrettyUri(leftURL)),
			Span: rightSpan,
		}
	}

	if right.IsOptional() && right.MediaContext() == nil {
		return left, nil
	}
	if left.IsOptional() && left.MediaContext() == nil {
		return right, nil
	}

	newExt := NewExtender(left.Extender().Selector, nil, false)
	leftSpan, err := left.Span()
	if err != nil {
		return nil, err
	}
	merged := &MergedExtension{
		BaseExtension: &BaseExtension{
			extender:   newExt,
			target:     left.Target(),
			span:       leftSpan,
			isOptional: true,
		},
		Left:  left,
		Right: right,
	}
	newExt.extension = merged
	if left.MediaContext() != nil {
		merged.BaseExtension.mediaContext = left.MediaContext()
	} else {
		merged.BaseExtension.mediaContext = right.MediaContext()
	}
	return merged, nil
}

// UnmergeExtensions flattens a possibly merged extension tree back to its
// leaf base extensions, so mandatory-branch reporting sees each original
// rule.
func UnmergeExtensions(e Extension) []Extension {
	if m, ok := e.(*MergedExtension); ok {
		var result []Extension
		result = append(result, UnmergeExtensions(m.Left)...)
		result = append(result, UnmergeExtensions(m.Right)...)
		return result
	}
	return []Extension{e}
}
