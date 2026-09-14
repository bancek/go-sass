// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/gamut_map_method.dart

import (
	"fmt"

	"github.com/bancek/go-sass/sasscommon"
)

// GamutMapMethod represents a method for mapping out-of-gamut colors into gamut.
//
// Matches Dart: GamutMapMethod (sealed base with clip and local-minde members).
type GamutMapMethod int

const (
	// GamutMapClip clamps each out-of-gamut channel to its min/max.
	//
	// Matches Dart: GamutMapMethod.clip. Visual quality is poor but the
	// result matches other clipping contexts.
	GamutMapClip GamutMapMethod = iota
	// GamutMapLocalMinde maps through Oklch using the deltaEOK difference
	// with the local-MINDE improvement.
	//
	// Matches Dart: GamutMapMethod.localMinde.
	GamutMapLocalMinde
)

// GamutMapMethodFromName parses a gamut map method from its Sass name.
//
// Matches Dart: GamutMapMethod.fromName. Unknown names raise a script error
// naming the offending argument when argumentName is set.
func GamutMapMethodFromName(name string) (GamutMapMethod, error) {
	switch name {
	case "clip":
		return GamutMapClip, nil
	case "local-minde":
		return GamutMapLocalMinde, nil
	default:
		return GamutMapClip, sasscommon.NewSassScriptException(fmt.Sprintf("Unknown gamut map method %q.", name), nil)
	}
}

// String returns the Sass name of the gamut map method.
//
// Matches Dart: GamutMapMethod.name/toString.
func (gmm GamutMapMethod) String() string {
	switch gmm {
	case GamutMapClip:
		return "clip"
	case GamutMapLocalMinde:
		return "local-minde"
	default:
		return fmt.Sprintf("GamutMapMethod(%d)", gmm)
	}
}
