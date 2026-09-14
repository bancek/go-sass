// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/math.dart

import (
	"math/rand"
)

// randomGen ports Dart's math.dart _random (an unseeded Random): the zero
// source is a deterministic placeholder replaced with fresh entropy at init.
var randomGen = rand.New(rand.NewSource(0))

func init() {
	randomGen = rand.New(rand.NewSource(rand.Int63()))
}
