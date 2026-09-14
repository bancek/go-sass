// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sasscommon

import "testing"

func TestFileSourceCharacterColumn(t *testing.T) {
	fs := NewFileSource([]byte("a\u25bcb\n"), nil)
	cases := []struct{ offset, want int }{
		{0, 0}, // 'a'
		{1, 1}, // start of the 3-byte ▼
		{4, 2}, // 'b'
		{5, 3}, // newline
	}
	for _, c := range cases {
		if got := fs.CharacterColumn(c.offset); got != c.want {
			t.Errorf("CharacterColumn(%d) = %d, want %d", c.offset, got, c.want)
		}
	}
}

func TestFrameForSpanCharacterColumn(t *testing.T) {
	fs := NewFileSource([]byte("a\u25bcb\n"), nil)
	span := NewSimpleFileSpan(fs, 4, 5)
	frame, err := FrameForSpan(span, "root stylesheet")
	if err != nil {
		t.Fatal(err)
	}
	if frame.Line != 1 || frame.Column != 3 {
		t.Errorf("frame = %+v, want line 1 column 3", frame)
	}
}
