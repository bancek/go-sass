// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/calculation.dart (warn callback type)

import "github.com/bancek/go-sass/deprecation"

// WarnCallback is used to surface deprecation warnings. The deprecationOpt
// parameter may be nil, matching Dart's optional [Deprecation?] parameter.
//
// Matches Dart: void Function(String message, [Deprecation? deprecationOpt])
type WarnCallback func(message string, deprecationOpt *deprecation.Deprecation) error
