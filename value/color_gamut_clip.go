// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package value

// dart-source: lib/src/value/color/gamut_map_method/clip.dart

import "github.com/bancek/go-sass/util"

// clipGamutMap clamps each channel to its valid range.
//
// Matches Dart: ClipGamutMap.map. The per-channel clamp helper is inlined:
// missing and polar-angle channels pass through untouched, and unbounded
// spaces skip clamping entirely (IsBounded gate).
func clipGamutMap(c *SassColor) *SassColor {
	chs := SpaceChannels(c.space)
	newCh := [3]float64{c.channel0, c.channel1, c.channel2}
	newMissing := c.missing

	for i := range 3 {
		ch := chs[i]
		if c.missing[i] || ch.IsPolarAngle {
			continue
		}
		if c.space.IsBounded() {
			newCh[i] = util.ClampLikeCss(newCh[i], ch.Min, ch.Max)
		}
	}

	return &SassColor{
		space:    c.space,
		channel0: newCh[0],
		channel1: newCh[1],
		channel2: newCh[2],
		alpha:    c.alpha,
		missing:  newMissing,
	}
}
