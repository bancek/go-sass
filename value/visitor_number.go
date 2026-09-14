// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/visitor/serialize.dart (_writeNumber section;
// the rounding and exponent helpers live in util/number_write.go)

import (
	"strings"

	"github.com/bancek/go-sass/util"
)

// writeNumber emits v in the shortest round-tripping decimal form: full
// precision in inspect mode, at most Precision fractional digits with
// ripple-carry rounding otherwise, integers without a ".0" suffix, and
// without a leading zero in compressed mode. It delegates to the shared
// util writer so plain number formatting stays identical everywhere.
//
// Matches Dart: _SerializeVisitor._writeNumber
func (sv *SerializeVisitor) writeNumber(v float64) {
	util.WriteNumberTo(sv.sb, v, sv.inspect, sv.isCompressed())
}

// writeNumberToString formats v as a CSS numeric string without touching
// the buffer. It is Go-only glue (Dart writes straight into _buffer or a
// capture buffer); no Dart counterpart exists.
//
// Matches Dart: _SerializeVisitor._writeNumber (buffer-less variant)
func (sv *SerializeVisitor) writeNumberToString(v float64) string {
	var buf strings.Builder
	util.WriteNumberTo(&buf, v, sv.inspect, sv.isCompressed())
	return buf.String()
}

// writeRounded rounds text (a number already free of exponent notation)
// to Precision digits after the decimal and writes the result. It
// delegates to the shared util writer.
//
// Matches Dart: _SerializeVisitor._writeRounded
func (sv *SerializeVisitor) writeRounded(text string) {
	util.WriteRoundedTo(sv.sb, text, sv.isCompressed())
}
