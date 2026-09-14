// Copyright (c) 2014, the Dart project authors.  Please see the AUTHORS file
// for details. All rights reserved. Use of this source code is governed by a
// BSD-style license that can be found in the LICENSE file.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

// Ported from dart string_scanner tests

package sasscommon

// dart-source: (external) package:string_scanner/test/span_scanner_test.dart, (external) package:string_scanner/test/string_scanner_test.dart, (external) package:string_scanner/test/line_scanner_test.dart, (external) package:string_scanner/test/error_test.dart

import (
	"net/url"
	"testing"
)

func checkScanError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("expected error")
	} else if _, ok := err.(*ScanError); !ok {
		t.Errorf("expected *ScanError, got %T: %v", err, err)
	}
}

func checkNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// -- empty source -----------------------------------------------------------

func TestEmptySource(t *testing.T) {
	s := NewSpanScanner([]byte{}, nil)

	t.Run("is done", func(t *testing.T) {
		if !s.IsDone() {
			t.Error("expected IsDone() = true")
		}
		checkNoError(t, s.ExpectDone())
	})

	t.Run("rest is empty", func(t *testing.T) {
		if s.Rest() != "" {
			t.Errorf("expected empty rest, got %q", s.Rest())
		}
	})

	t.Run("position is zero", func(t *testing.T) {
		if s.Position() != 0 {
			t.Errorf("expected position 0, got %d", s.Position())
		}
	})

	t.Run("readChar returns error", func(t *testing.T) {
		_, err := s.ReadChar()
		checkScanError(t, err)
	})

	t.Run("peekChar returns -1", func(t *testing.T) {
		if ch := s.PeekChar(0); ch >= 0 {
			t.Errorf("expected -1, got %d", ch)
		}
	})

	t.Run("scanChar returns false", func(t *testing.T) {
		if s.ScanChar('a') {
			t.Error("expected ScanChar to return false")
		}
	})

	t.Run("expectChar returns error", func(t *testing.T) {
		err := s.ExpectChar('a')
		checkScanError(t, err)
	})

	t.Run("substring returns empty string", func(t *testing.T) {
		if s.Substring(0, nil) != "" {
			t.Errorf("expected empty substring, got %q", s.Substring(0, nil))
		}
	})

	t.Run("SetPosition(1) returns error", func(t *testing.T) {
		err := s.SetPosition(1)
		checkScanError(t, err)
	})

	t.Run("SetPosition(-1) returns error", func(t *testing.T) {
		err := s.SetPosition(-1)
		checkScanError(t, err)
	})
}

// -- beginning of source ----------------------------------------------------

func TestBeginningOfSource(t *testing.T) {
	source := []byte("foo bar")
	s := NewSpanScanner(source, nil)

	t.Run("is not done", func(t *testing.T) {
		if s.IsDone() {
			t.Error("expected IsDone() = false")
		}
		checkScanError(t, s.ExpectDone())
	})

	t.Run("rest is the whole source", func(t *testing.T) {
		if s.Rest() != "foo bar" {
			t.Errorf("expected 'foo bar', got %q", s.Rest())
		}
	})

	t.Run("position is zero", func(t *testing.T) {
		if s.Position() != 0 {
			t.Errorf("expected position 0, got %d", s.Position())
		}
	})

	t.Run("readChar returns first byte and moves forward", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != 'f' {
			t.Errorf("expected 'f' (%d), got %d", 'f', ch)
		}
		if s2.Position() != 1 {
			t.Errorf("expected position 1, got %d", s2.Position())
		}
	})

	t.Run("peekChar returns first byte", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		ch := s2.PeekChar(0)
		if ch != 'f' {
			t.Errorf("expected 'f', got %d", ch)
		}
		if s2.Position() != 0 {
			t.Errorf("expected position unchanged 0, got %d", s2.Position())
		}
	})

	t.Run("peekChar with offset returns nth byte", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		ch := s2.PeekChar(4)
		if ch != 'b' {
			t.Errorf("expected 'b' (%d), got %d", 'b', ch)
		}
		if s2.Position() != 0 {
			t.Errorf("expected position unchanged 0, got %d", s2.Position())
		}
	})

	t.Run("matching scanChar returns true and moves forward", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		if !s2.ScanChar('f') {
			t.Error("expected ScanChar('f') = true")
		}
		if s2.Position() != 1 {
			t.Errorf("expected position 1, got %d", s2.Position())
		}
	})

	t.Run("non-matching scanChar returns false", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		if s2.ScanChar('x') {
			t.Error("expected ScanChar('x') = false")
		}
		if s2.Position() != 0 {
			t.Errorf("expected position unchanged 0, got %d", s2.Position())
		}
	})

	t.Run("matching expectChar succeeds", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.ExpectChar('f'))
		if s2.Position() != 1 {
			t.Errorf("expected position 1, got %d", s2.Position())
		}
	})

	t.Run("non-matching expectChar returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		err := s2.ExpectChar('x')
		checkScanError(t, err)
	})

	t.Run("matching Expect succeeds", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		if s2.Position() != 3 {
			t.Errorf("expected position 3, got %d", s2.Position())
		}
		if s2.Rest() != " bar" {
			t.Errorf("expected rest ' bar', got %q", s2.Rest())
		}
	})

	t.Run("non-matching Expect returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		err := s2.Expect("bar")
		checkScanError(t, err)
	})

	t.Run("substring from beginning is empty", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		if s2.Substring(0, nil) != "" {
			t.Errorf("expected empty, got %q", s2.Substring(0, nil))
		}
	})

	t.Run("substring with custom end", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		if s2.Substring(0, new(3)) != "foo" {
			t.Errorf("expected 'foo', got %q", s2.Substring(0, new(3)))
		}
	})

	t.Run("substring with source length returns whole source", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		sourceLen := len(source)
		if s2.Substring(0, &sourceLen) != "foo bar" {
			t.Errorf("expected 'foo bar', got %q", s2.Substring(0, &sourceLen))
		}
	})

	t.Run("SetPosition moves cursor forward", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.SetPosition(1))
		if s2.Position() != 1 {
			t.Errorf("expected position 1, got %d", s2.Position())
		}
		if s2.Rest() != "oo bar" {
			t.Errorf("expected rest 'oo bar', got %q", s2.Rest())
		}
		checkNoError(t, s2.Expect("oo "))
		if s2.Rest() != "bar" {
			t.Errorf("expected rest 'bar', got %q", s2.Rest())
		}
	})

	t.Run("SetPosition beyond source returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		err := s2.SetPosition(8)
		checkScanError(t, err)
	})

	t.Run("SetPosition(-1) returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		err := s2.SetPosition(-1)
		checkScanError(t, err)
	})
}

// -- after Expect -----------------------------------------------------------

func TestAfterExpect(t *testing.T) {
	source := []byte("foo bar")

	t.Run("readChar returns next byte", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != ' ' {
			t.Errorf("expected ' ' (%d), got %d", ' ', ch)
		}
		if s2.Position() != 4 {
			t.Errorf("expected position 4, got %d", s2.Position())
		}
	})

	t.Run("matching scanChar returns true", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		if !s2.ScanChar(' ') {
			t.Error("expected ScanChar(' ') = true")
		}
		if s2.Position() != 4 {
			t.Errorf("expected position 4, got %d", s2.Position())
		}
	})

	t.Run("matching expectChar succeeds", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		checkNoError(t, s2.ExpectChar(' '))
		if s2.Position() != 4 {
			t.Errorf("expected position 4, got %d", s2.Position())
		}
	})
}

// -- end of source ----------------------------------------------------------

func TestEndOfSource(t *testing.T) {
	source := []byte("foo bar")
	s := NewSpanScanner(source, nil)
	checkNoError(t, s.Expect("foo bar"))

	t.Run("is done", func(t *testing.T) {
		if !s.IsDone() {
			t.Error("expected IsDone() = true")
		}
		checkNoError(t, s.ExpectDone())
	})

	t.Run("rest is empty", func(t *testing.T) {
		if s.Rest() != "" {
			t.Errorf("expected empty rest, got %q", s.Rest())
		}
	})

	t.Run("position is at end", func(t *testing.T) {
		if s.Position() != 7 {
			t.Errorf("expected position 7, got %d", s.Position())
		}
	})

	t.Run("readChar returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		_, err := s2.ReadChar()
		checkScanError(t, err)
	})

	t.Run("peekChar returns -1", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		if ch := s2.PeekChar(0); ch >= 0 {
			t.Errorf("expected -1, got %d", ch)
		}
	})

	t.Run("scanChar returns false", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		if s2.ScanChar('f') {
			t.Error("expected ScanChar to return false")
		}
	})

	t.Run("expectChar returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		err := s2.ExpectChar('f')
		checkScanError(t, err)
	})

	t.Run("substring from beginning returns whole source", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		if s2.Substring(0, nil) != "foo bar" {
			t.Errorf("expected 'foo bar', got %q", s2.Substring(0, nil))
		}
	})

	t.Run("substring with custom start", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		if s2.Substring(4, nil) != "bar" {
			t.Errorf("expected 'bar', got %q", s2.Substring(4, nil))
		}
	})

	t.Run("substring with custom start and end", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		if s2.Substring(3, new(5)) != " b" {
			t.Errorf("expected ' b', got %q", s2.Substring(3, new(5)))
		}
	})

	t.Run("SetPosition moves backward", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		checkNoError(t, s2.SetPosition(1))
		if s2.Position() != 1 {
			t.Errorf("expected position 1, got %d", s2.Position())
		}
		if s2.Rest() != "oo bar" {
			t.Errorf("expected rest 'oo bar', got %q", s2.Rest())
		}
		checkNoError(t, s2.Expect("oo "))
		if s2.Rest() != "bar" {
			t.Errorf("expected rest 'bar', got %q", s2.Rest())
		}
	})

	t.Run("SetPosition beyond source returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		err := s2.SetPosition(8)
		checkScanError(t, err)
	})

	t.Run("SetPosition(-1) returns error", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo bar"))
		err := s2.SetPosition(-1)
		checkScanError(t, err)
	})
}

// -- Expect with string containing % ----------------------------------------

func TestExpectPercent(t *testing.T) {
	t.Run("non-matching Expect with %% in string does not garble error", func(t *testing.T) {
		source := []byte("foo")
		s := NewSpanScanner(source, nil)
		err := s.Expect("%foo")
		checkScanError(t, err)
		if scanErr, ok := err.(*ScanError); ok {
			if scanErr.Message != `expected "%foo".` {
				t.Errorf("expected message %q, got %q", `expected "%foo".`, scanErr.Message)
			}
		}
	})
}

// -- line / column ----------------------------------------------------------

func TestLineColumn(t *testing.T) {
	source := []byte("foo\nbar\r\nbaz")
	s := NewSpanScanner(source, nil)

	t.Run("begins with line 0 column 0", func(t *testing.T) {
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 0 {
			t.Errorf("expected column 0, got %d", s.Column())
		}
	})

	t.Run("consuming no newlines increases column", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		if s2.Line() != 0 {
			t.Errorf("expected line 0, got %d", s2.Line())
		}
		if s2.Column() != 3 {
			t.Errorf("expected column 3, got %d", s2.Column())
		}
	})

	t.Run("readChar on non-newline increases column", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		_, err := s2.ReadChar()
		checkNoError(t, err)
		if s2.Line() != 0 {
			t.Errorf("expected line 0, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("LF resets column and increases line", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo\nba"))
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 2 {
			t.Errorf("expected column 2, got %d", s2.Column())
		}
	})

	t.Run("multiple LFs", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo\nbar\r\nb"))
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("CR LF increases line only after LF", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo\nbar\r"))
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 4 {
			t.Errorf("expected column 4, got %d", s2.Column())
		}

		checkNoError(t, s2.Expect("\nb"))
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("CR not followed by LF increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\nbar\rbaz"), nil)
		checkNoError(t, s2.Expect("foo\nbar\r"))
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}

		checkNoError(t, s2.Expect("b"))
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("CR at end increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\nbar\r"), nil)
		checkNoError(t, s2.Expect("foo\nbar\r"))
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
		if !s2.IsDone() {
			t.Error("expected IsDone() = true")
		}
	})

	t.Run("mix of CR, LF, CR+LF", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("0\n1\r2\r\n3"), nil)
		checkNoError(t, s2.Expect("0\n1\r2\r\n3"))
		if s2.Line() != 3 {
			t.Errorf("expected line 3, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("readChar LF resets column and increases line", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != '\n' {
			t.Errorf("expected LF (%d), got %d", '\n', ch)
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("readChar CR LF increases line only after LF", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\r\nbar"), nil)
		checkNoError(t, s2.Expect("foo"))
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != '\r' {
			t.Errorf("expected CR (%d), got %d", '\r', ch)
		}
		if s2.Line() != 0 {
			t.Errorf("expected line 0, got %d", s2.Line())
		}
		if s2.Column() != 4 {
			t.Errorf("expected column 4, got %d", s2.Column())
		}

		ch, err = s2.ReadChar()
		checkNoError(t, err)
		if ch != '\n' {
			t.Errorf("expected LF (%d), got %d", '\n', ch)
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("readChar CR not followed by LF increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\nbar\rbaz"), nil)
		checkNoError(t, s2.Expect("foo\nbar"))
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != '\r' {
			t.Errorf("expected CR (%d), got %d", '\r', ch)
		}
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("readChar CR at end increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\nbar\r"), nil)
		checkNoError(t, s2.Expect("foo\nbar"))
		ch, err := s2.ReadChar()
		checkNoError(t, err)
		if ch != '\r' {
			t.Errorf("expected CR (%d), got %d", '\r', ch)
		}
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("scanChar non-newline increases column", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		if !s2.ScanChar('f') {
			t.Error("expected ScanChar('f') = true")
		}
		if s2.Line() != 0 {
			t.Errorf("expected line 0, got %d", s2.Line())
		}
		if s2.Column() != 1 {
			t.Errorf("expected column 1, got %d", s2.Column())
		}
	})

	t.Run("scanChar LF resets column and increases line", func(t *testing.T) {
		s2 := NewSpanScanner(source, nil)
		checkNoError(t, s2.Expect("foo"))
		if !s2.ScanChar('\n') {
			t.Error("expected ScanChar('\\n') = true")
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("scanChar CR LF increases line only after LF", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s2.Expect("foo\nbar"))
		if !s2.ScanChar('\r') {
			t.Error("expected ScanChar('\\r') = true")
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 4 {
			t.Errorf("expected column 4, got %d", s2.Column())
		}
		if !s2.ScanChar('\n') {
			t.Error("expected ScanChar('\\n') = true")
		}
		if s2.Line() != 2 {
			t.Errorf("expected line 2, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("scanChar CR not followed by LF increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\rbar"), nil)
		checkNoError(t, s2.Expect("foo"))
		if !s2.ScanChar('\r') {
			t.Error("expected ScanChar('\\r') = true")
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})

	t.Run("scanChar CR at end increases line", func(t *testing.T) {
		s2 := NewSpanScanner([]byte("foo\r"), nil)
		checkNoError(t, s2.Expect("foo"))
		if !s2.ScanChar('\r') {
			t.Error("expected ScanChar('\\r') = true")
		}
		if s2.Line() != 1 {
			t.Errorf("expected line 1, got %d", s2.Line())
		}
		if s2.Column() != 0 {
			t.Errorf("expected column 0, got %d", s2.Column())
		}
	})
}

// -- SetPosition ------------------------------------------------------------

func TestSetPosition(t *testing.T) {
	t.Run("forward through LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\nbaz"), nil)
		checkNoError(t, s.SetPosition(9))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward from non-zero through LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\nbaz"), nil)
		checkNoError(t, s.Expect("fo"))
		checkNoError(t, s.SetPosition(9))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward through CR LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\r\nbar\r\nbaz"), nil)
		checkNoError(t, s.SetPosition(11))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward through CR not followed by LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\rbar\rbaz"), nil)
		checkNoError(t, s.SetPosition(9))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward through CR at end", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\rbar\r"), nil)
		checkNoError(t, s.SetPosition(8))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 0 {
			t.Errorf("expected column 0, got %d", s.Column())
		}
	})

	t.Run("forward through mix of CR, LF, CR+LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("0\n1\r2\r\n3"), nil)
		checkNoError(t, s.SetPosition(len("0\n1\r2\r\n3")))
		if s.Line() != 3 {
			t.Errorf("expected line 3, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward through no newlines sets column", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.SetPosition(2))
		if s.Line() != 0 { // "fo"
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward through LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\nbaz"))
		checkNoError(t, s.SetPosition(2))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward through CR LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\r\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\r\nbar\r\nbaz"))
		checkNoError(t, s.SetPosition(2))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward through CR not followed by LFs", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\rbar\rbaz"), nil)
		checkNoError(t, s.Expect("foo\rbar\rbaz"))
		checkNoError(t, s.SetPosition(2))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward through CR at end", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\rbar\r"), nil)
		checkNoError(t, s.Expect("foo\rbar\r"))
		checkNoError(t, s.SetPosition(2))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward through mix of CR, LF, CR+LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("0\n1\r2\r\n3"), nil)
		checkNoError(t, s.Expect("0\n1\r2\r\n3"))
		checkNoError(t, s.SetPosition(1))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("backward through no newlines sets column", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r\nbaz"))
		checkNoError(t, s.SetPosition(10))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("forward halfway through CR LF does not count as line", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.SetPosition(8))
		if s.Line() != 1 {
			t.Errorf("expected line 1, got %d", s.Line())
		}
		if s.Column() != 4 {
			t.Errorf("expected column 4, got %d", s.Column())
		}
	})

	t.Run("forward from halfway through CR LF counts as line", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r"))
		checkNoError(t, s.SetPosition(11))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 2 {
			t.Errorf("expected column 2, got %d", s.Column())
		}
	})

	t.Run("backward to between CR LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r\nbaz"))
		checkNoError(t, s.SetPosition(8))
		if s.Line() != 1 {
			t.Errorf("expected line 1, got %d", s.Line())
		}
		if s.Column() != 4 {
			t.Errorf("expected column 4, got %d", s.Column())
		}
	})

	t.Run("backward from between CR LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r"))
		checkNoError(t, s.SetPosition(5))
		if s.Line() != 1 {
			t.Errorf("expected line 1, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("backward from between CR LF to before CR LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r"))
		checkNoError(t, s.SetPosition(1))
		if s.Line() != 0 {
			t.Errorf("expected line 0, got %d", s.Line())
		}
		if s.Column() != 1 {
			t.Errorf("expected column 1, got %d", s.Column())
		}
	})

	t.Run("backward to after CR LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r\nbaz"))
		checkNoError(t, s.SetPosition(9))
		if s.Line() != 2 {
			t.Errorf("expected line 2, got %d", s.Line())
		}
		if s.Column() != 0 {
			t.Errorf("expected column 0, got %d", s.Column())
		}
	})

	t.Run("backward to before CR LF", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
		checkNoError(t, s.Expect("foo\nbar\r\nbaz"))
		checkNoError(t, s.SetPosition(7))
		if s.Line() != 1 {
			t.Errorf("expected line 1, got %d", s.Line())
		}
		if s.Column() != 3 {
			t.Errorf("expected column 3, got %d", s.Column())
		}
	})
}

// -- state save/restore -----------------------------------------------------

func TestState(t *testing.T) {
	s := NewSpanScanner([]byte("foo\nbar\r\nbaz"), nil)
	checkNoError(t, s.Expect("foo\nb"))

	state := s.State()
	if s.Rest() != "ar\r\nbaz" {
		t.Errorf("expected rest 'ar\\r\\nbaz', got %q", s.Rest())
	}
	if s.Line() != 1 {
		t.Errorf("expected line 1, got %d", s.Line())
	}
	if s.Column() != 1 {
		t.Errorf("expected column 1, got %d", s.Column())
	}

	checkNoError(t, s.Expect("ar\r\nba"))
	s.SetState(state)
	if s.Rest() != "ar\r\nbaz" {
		t.Errorf("expected rest 'ar\\r\\nbaz', got %q", s.Rest())
	}
	if s.Line() != 1 {
		t.Errorf("expected line 1, got %d", s.Line())
	}
	if s.Column() != 1 {
		t.Errorf("expected column 1, got %d", s.Column())
	}
}

// -- SpanFrom / SpanFromPosition / EmptySpan / Location ---------------------

func TestSpanFrom(t *testing.T) {
	s := NewSpanScanner([]byte("foo\nbar\nbaz"), mustParseURL(t, "source"))
	checkNoError(t, s.Expect("fo"))
	state := s.State()
	checkNoError(t, s.Expect("o\nba"))
	checkNoError(t, s.Expect("r\nba"))

	span := s.SpanFrom(state)
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "o\nbar\nba" {
		t.Errorf("expected text 'o\\nbar\\nba', got %q", text)
	}
}

func TestSpanFromPosition(t *testing.T) {
	s := NewSpanScanner([]byte("foo\nbar\nbaz"), mustParseURL(t, "source"))
	checkNoError(t, s.Expect("fo"))
	start := s.Position()
	checkNoError(t, s.Expect("o\nba"))
	checkNoError(t, s.Expect("r\nba"))

	span := s.SpanFromPosition(start)
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "o\nbar\nba" {
		t.Errorf("expected text 'o\\nbar\\nba', got %q", text)
	}
}

func TestEmptySpan(t *testing.T) {
	s := NewSpanScanner([]byte("foo\nbar\nbaz"), mustParseURL(t, "source"))
	checkNoError(t, s.Expect("foo\nba"))

	span := s.EmptySpan()
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	startLoc, err := span.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := span.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != endLoc.Offset {
		t.Errorf("expected start offset %d == end offset %d", startLoc.Offset, endLoc.Offset)
	}
	if startLoc.Line != 1 {
		t.Errorf("expected line 1, got %d", startLoc.Line)
	}
	if startLoc.Column != 2 {
		t.Errorf("expected column 2, got %d", startLoc.Column)
	}
}

func TestLocation(t *testing.T) {
	s := NewSpanScanner([]byte("foo\nbar"), nil)
	loc := s.Location()
	if loc.Offset != 0 {
		t.Errorf("expected offset 0, got %d", loc.Offset)
	}
	if loc.Line != 0 {
		t.Errorf("expected line 0, got %d", loc.Line)
	}
	if loc.Column != 0 {
		t.Errorf("expected column 0, got %d", loc.Column)
	}

	checkNoError(t, s.Expect("foo\n"))
	loc = s.Location()
	if loc.Offset != 4 {
		t.Errorf("expected offset 4, got %d", loc.Offset)
	}
	if loc.Line != 1 {
		t.Errorf("expected line 1, got %d", loc.Line)
	}
	if loc.Column != 0 {
		t.Errorf("expected column 0, got %d", loc.Column)
	}
}

// -- Error ------------------------------------------------------------------

func TestErrorDefaults(t *testing.T) {
	t.Run("defaults to length 0", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz"), nil)
		checkNoError(t, s.Expect("foo "))
		scanErr := s.Error("oh no!", 1, -1)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "" {
			t.Errorf("expected empty span text, got %q", text)
		}
	})

	t.Run("defaults position to current when negative", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz"), nil)
		checkNoError(t, s.Expect("foo "))
		scanErr := s.Error("oh no!", -1, 3)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "bar" {
			t.Errorf("expected span text 'bar', got %q", text)
		}
	})

	t.Run("supports earlier position", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz"), nil)
		checkNoError(t, s.Expect("foo "))
		scanErr := s.Error("oh no!", 1, 2)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "oo" {
			t.Errorf("expected span text 'oo', got %q", text)
		}
	})

	t.Run("supports position on previous line", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz\ndo re mi\nearth fire water"), nil)
		checkNoError(t, s.Expect("foo bar baz\ndo re mi\nearth"))
		scanErr := s.Error("oh no!", 15, 2)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "re" {
			t.Errorf("expected span text 're', got %q", text)
		}
	})

	t.Run("supports multiline length", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz\ndo re mi\nearth fire water"), nil)
		checkNoError(t, s.Expect("foo bar baz\ndo re mi\nearth"))
		scanErr := s.Error("oh no!", 8, 8)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "baz\ndo r" {
			t.Errorf("expected span text 'baz\\ndo r', got %q", text)
		}
	})

	t.Run("supports position after current", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz"), nil)
		scanErr := s.Error("oh no!", 4, 3)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "bar" {
			t.Errorf("expected span text 'bar', got %q", text)
		}
	})

	t.Run("supports length of zero", func(t *testing.T) {
		s := NewSpanScanner([]byte("foo bar baz"), nil)
		scanErr := s.Error("oh no!", 4, 0)
		text, err := scanErr.(*ScanError).Span.SpanText()
		if err != nil {
			t.Fatal(err)
		}
		if text != "" {
			t.Errorf("expected empty span text, got %q", text)
		}
	})
}

func TestErrorSourceURL(t *testing.T) {
	s := NewSpanScanner([]byte("foo bar baz"), mustParseURL(t, "source"))
	checkNoError(t, s.Expect("foo "))
	expectedFile := mustParseURL(t, "source").String()

	scanErr := s.Error("oh no!", -1, 3)
	sourceURL, err := scanErr.(*ScanError).Span.SourceURL()
	if err != nil {
		t.Fatal(err)
	}
	if sourceURL.String() != expectedFile {
		t.Errorf("expected file %q, got %q", expectedFile, sourceURL.String())
	}
}

// -- helpers ----------------------------------------------------------------

func mustParseURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
