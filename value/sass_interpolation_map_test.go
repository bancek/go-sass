package value

import (
	"net/url"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

func TestInterpolationMap_MapSpan(t *testing.T) {
	source := `a.#{expr}.b`

	// Create FileSource with the full source text
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))

	// Source: a.#{expr}.b
	//         0123456789 10 (11 chars)
	// # at 2, { at 3, e at 4, r at 7, } at 8
	varSpan := sasscommon.NewFileSpan(fs, 4, 8)

	expr := NewVariableExpression("expr", varSpan, nil)

	interp, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, &varSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Generated text: "a.VALUE.b" (10 chars).
	// targetOffsets[0] = 2 (after "a.")
	// targetOffsets[1] = 7 (after "a.VALUE")
	targetOffsets := []int{2, 7}

	m, err := NewInterpolationMap(interp, targetOffsets)
	if err != nil {
		t.Fatal(err)
	}

	// The generated text is "a.VALUE.b". The expression maps to offsets 2-7.
	// So the generated span starts at 2 (after "a.") and ends at 7 (before ".b").
	// Use a non-nil URL so isMapped distinguishes it from the source file.
	genURL, _ := url.Parse("generated:///output")
	genFs := sasscommon.NewFileSource([]byte("a.VALUE.b"), genURL)
	generatedSpan := sasscommon.NewFileSpan(genFs, 2, 7)

	mapped, err := m.MapSpan(generatedSpan)
	if err != nil {
		t.Fatal(err)
	}

	fs2, ok := mapped.(sasscommon.FileSpan)
	if !ok {
		t.Fatal("MapSpan did not return a FileSpan")
	}
	if fs2 == generatedSpan {
		t.Error("MapSpan returned the same span — mapping is a no-op")
	}
	// expandLeft scans from expr start (4) backward to find # at offset 2.
	// expandRight scans from expr end (8) forward to find } at offset 9.
	// Result: span [2, 9) covering #{expr}.
	startLoc, err := fs2.StartLocation()
	if err != nil {
		t.Fatal(err)
	}
	endLoc, err := fs2.EndLocation()
	if err != nil {
		t.Fatal(err)
	}
	if startLoc.Offset != 2 || endLoc.Offset != 9 {
		t.Errorf("MapSpan returned span at offset %d-%d, want 2-9",
			startLoc.Offset, endLoc.Offset)
	}
}

func TestInterpolationMap_MapException(t *testing.T) {
	// Source: .foo#{expr}.bar
	//         0         1         2
	//         0123456789012345678901234
	// . at 0, # at 4, { at 5, e at 6, r at 9, } at 10, .bar at 11-14
	source := `.foo#{expr}.bar`
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))

	varSpan := sasscommon.NewFileSpan(fs, 6, 10)

	expr := NewVariableExpression("expr", varSpan, nil)

	interp, err := NewInterpolation(
		[]any{".foo", expr, ".bar"},
		[]*sasscommon.FileSpan{nil, &varSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	// generated text: ".fooVALUE.bar" — targetOffsets after each segment
	targetOffsets := []int{4, 9}
	m, err := NewInterpolationMap(interp, targetOffsets)
	if err != nil {
		t.Fatal(err)
	}

	// Error at generated offset 4-5 ("V" in "a.VALUE.b")
	// Use a non-nil URL so isMapped distinguishes it from the source file.
	genURL, _ := url.Parse("generated:///output")
	genFs := sasscommon.NewFileSource([]byte("a.VALUE.b"), genURL)
	generatedSpan := sasscommon.NewFileSpan(genFs, 4, 5)
	ssfErr := &sasscommon.SourceSpanFormatException{Message: "expected selector", Span: generatedSpan}

	mapped := m.MapException(ssfErr)
	if mapped == ssfErr {
		t.Error("MapException returned the same error — mapping is a no-op")
	}
}

func TestInterpolationMapNewValidation(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("x"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 1)
	// interpolation has 1 content element, so targetOffsets should have 0 elements
	interp, err := NewInterpolation(
		[]any{"x"},
		[]*sasscommon.FileSpan{nil},
		span,
	)
	if err != nil {
		t.Fatal(err)
	}

	// targetOffsets must have length = contents.len() - 1 = 0
	_, err = NewInterpolationMap(interp, []int{})
	if err != nil {
		t.Fatal(err)
	}

	// Wrong length
	_, err = NewInterpolationMap(interp, []int{1})
	if err == nil {
		t.Fatal("expected error for wrong targetOffsets length")
	}
}

func TestInterpolationMapEmptyContents(t *testing.T) {
	fs := sasscommon.NewFileSource([]byte("hello"), nil)
	span := sasscommon.NewSimpleFileSpan(fs, 0, 5)
	interp, err := NewInterpolation([]any{}, []*sasscommon.FileSpan{}, span)
	if err != nil {
		t.Fatal(err)
	}

	m, err := NewInterpolationMap(interp, []int{})
	if err != nil {
		t.Fatal(err)
	}

	genURL, _ := url.Parse("generated:///output")
	genFs := sasscommon.NewFileSource([]byte("output"), genURL)
	genSpan := sasscommon.NewFileSpan(genFs, 2, 4)

	mapped, err := m.MapSpan(genSpan)
	if err != nil {
		t.Fatal(err)
	}
	// Empty interpolation: MapSpan returns the interpolation span
	if fs2, ok := mapped.(sasscommon.FileSpan); ok {
		startLoc, _ := fs2.StartLocation()
		endLoc, _ := fs2.EndLocation()
		if startLoc.Offset != 0 || endLoc.Offset != 5 {
			t.Errorf("MapSpan returned [%d, %d), want [0, 5)", startLoc.Offset, endLoc.Offset)
		}
	} else {
		t.Fatal("MapSpan did not return a FileSpan for empty interpolation")
	}
}

func TestInterpolationMapAlreadyMapped(t *testing.T) {
	source := "a.#{expr}.b"
	fs := sasscommon.NewFileSource([]byte(source), nil)
	fileSpan := sasscommon.NewSimpleFileSpan(fs, 0, len(source))
	varSpan := sasscommon.NewFileSpan(fs, 4, 9)
	// #{ spans 2-10
	interpExprSpan := sasscommon.NewFileSpan(fs, 2, 10)

	expr := NewVariableExpression("expr", varSpan, nil)

	interp, err := NewInterpolation(
		[]any{"a.", expr, ".b"},
		[]*sasscommon.FileSpan{nil, &interpExprSpan, nil},
		fileSpan,
	)
	if err != nil {
		t.Fatal(err)
	}

	targetOffsets := []int{2, 7}
	m, err := NewInterpolationMap(interp, targetOffsets)
	if err != nil {
		t.Fatal(err)
	}

	// Create a span from the same file source — should be treated as already mapped
	sameSourceSpan := sasscommon.NewFileSpan(fs, 0, 1)
	mapped, err := m.MapSpan(sameSourceSpan)
	if err != nil {
		t.Fatal(err)
	}
	if mapped != sameSourceSpan {
		t.Error("MapSpan should return unchanged span when already mapped")
	}
}
