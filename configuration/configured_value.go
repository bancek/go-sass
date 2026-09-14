// Copyright 2019 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package configuration

// dart-source: lib/src/configured_value.dart

import (
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// ConfiguredValue is a variable value that's been configured for a Configuration.
type ConfiguredValue struct {
	// The value of the variable.
	Value value.Value

	// The span where the variable's configuration was written, or nil if this
	// value was configured implicitly.
	ConfigurationSpan *sasscommon.FileSpan

	// The AstNode where the variable's value originated.
	AssignmentNode sasscommon.AstNode
}

// NewConfiguredValueExplicit creates a variable value that's been configured
// explicitly with a `with` clause.
func NewConfiguredValueExplicit(val value.Value, configurationSpan *sasscommon.FileSpan, assignmentNode sasscommon.AstNode) *ConfiguredValue {
	return &ConfiguredValue{
		Value:             val,
		ConfigurationSpan: configurationSpan,
		AssignmentNode:    assignmentNode,
	}
}

// NewConfiguredValueImplicit creates a variable value that's implicitly
// configured by setting a variable prior to an @import of a file that contains
// a @forward.
func NewConfiguredValueImplicit(val value.Value, assignmentNode sasscommon.AstNode) *ConfiguredValue {
	return &ConfiguredValue{
		Value:          val,
		AssignmentNode: assignmentNode,
	}
}

// String returns the configured value's Sass string form, delegating to the
// underlying value.
//
// Matches Dart: ConfiguredValue.toString forwarding to value.toString.
func (cv *ConfiguredValue) String() (string, error) { return cv.Value.String() }
