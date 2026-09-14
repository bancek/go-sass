// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package functions

// dart-source: lib/src/functions/string.dart

import (
	"math/rand"
)

// stringRandom and prevUniqueID port Dart's string.dart _random and
// _previousUniqueId: the generator behind unique-id, with the previous ID
// starting at a random point in the 36^6 ID space. The zero sources are
// deterministic placeholders reseeded with fresh entropy at init.
var stringRandom = rand.New(rand.NewSource(0))
var prevUniqueID int64

func init() {
	stringRandom = rand.New(rand.NewSource(rand.Int63()))
	prevUniqueID = stringRandom.Int63n(36 * 36 * 36 * 36 * 36 * 36)
}
