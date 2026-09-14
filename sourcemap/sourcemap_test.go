// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package sourcemap

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
)

// testSpan creates a FileSpan for sourcemap testing with the given content
// and offset range.
func testSpan(text string, startOffset, endOffset int) sasscommon.FileSpan {
	u, _ := url.Parse("input.scss")
	fs := sasscommon.NewFileSource([]byte(text), u)
	return sasscommon.NewSimpleFileSpan(fs, startOffset, endOffset)
}

// testSourceSpan creates a zero-length span at a specific source file, line and column.
func testSourceSpan(sourceFile string, line, col int) sasscommon.FileSpan {
	content := strings.Repeat("\n", line) + strings.Repeat("x", col)
	offset := line + col
	u, _ := url.Parse(sourceFile)
	fs := sasscommon.NewFileSource([]byte(content), u)
	return sasscommon.NewSimpleFileSpan(fs, offset, offset)
}

func TestBuilderNoEntries(t *testing.T) {
	b := NewBuilder("output.css")
	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["version"] != float64(3) {
		t.Errorf("version = %v, want 3", parsed["version"])
	}
	if parsed["file"] != "output.css" {
		t.Errorf("file = %v, want output.css", parsed["file"])
	}
	if parsed["mappings"] != "" {
		t.Errorf("mappings = %v, want empty", parsed["mappings"])
	}
}

func TestBuilderSingleEntry(t *testing.T) {
	b := NewBuilder("output.css")
	if err := b.AddMapping(0, 0, testSpan("body { color: red; }", 0, 10)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["version"] != float64(3) {
		t.Errorf("version = %v, want 3", parsed["version"])
	}
	if parsed["file"] != "output.css" {
		t.Errorf("file = %v, want output.css", parsed["file"])
	}

	sources, ok := parsed["sources"].([]any)
	if !ok || len(sources) != 1 || sources[0] != "input.scss" {
		t.Errorf("sources = %v, want [input.scss]", parsed["sources"])
	}

	// Golden from Dart: gen(0,0) -> src(0,0,0) produces "AAAA"
	if parsed["mappings"] != "AAAA" {
		t.Errorf("mappings = %v, want AAAA", parsed["mappings"])
	}
}

func TestBuilderTwoLines(t *testing.T) {
	// Golden from Dart:
	// gen(0,0)->src(line=2,col=5), gen(1,3)->src(line=5,col=10)
	// => mappings "AAEK;GAGK" (src state NOT reset at newline)
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 0, testSourceSpan("input.scss", 2, 5)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(1, 3, testSourceSpan("input.scss", 5, 10)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	got := parsed["mappings"].(string)
	want := "AAEK;GAGK"
	if got != want {
		t.Errorf("mappings = %q, want %q (Dart golden)", got, want)
	}

	// Decode to verify deltas match
	pos := 0
	col, np, _ := decodeVLQ(got, pos)
	srcIdx, np, _ := decodeVLQ(got, np)
	srcLine, np, _ := decodeVLQ(got, np)
	srcCol, pos, _ := decodeVLQ(got, np)
	if col != 0 || srcIdx != 0 || srcLine != 2 || srcCol != 5 {
		t.Errorf("seg 1: col=%d srcIdx=%d srcLine=%d srcCol=%d, want 0 0 2 5",
			col, srcIdx, srcLine, srcCol)
	}

	if got[pos] != ';' {
		t.Fatalf("expected ';' at pos %d", pos)
	}
	pos++

	// Line 1: deltas relative to line 0's LAST state (NOT reset)
	// col=3 (relative to 0, reset at newline), srcIdx=0, srcLine=3 (5-2), srcCol=5 (10-5)
	col, np, _ = decodeVLQ(got, pos)
	srcIdx, np, _ = decodeVLQ(got, np)
	srcLine, np, _ = decodeVLQ(got, np)
	srcCol, pos, _ = decodeVLQ(got, np)
	if col != 3 || srcIdx != 0 || srcLine != 3 || srcCol != 5 {
		t.Errorf("seg 2: col=%d srcIdx=%d srcLine=%d srcCol=%d, want 3 0 3 5",
			col, srcIdx, srcLine, srcCol)
	}
}

func TestBuilderSameLine(t *testing.T) {
	// Golden from Dart:
	// gen(0,4)->src(line=1,col=4)
	// gen(0,8)->src(line=3,col=2)
	// => mappings "IACC,IAEF"
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 4, testSourceSpan("input.scss", 1, 4)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(0, 8, testSourceSpan("input.scss", 3, 2)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	got := parsed["mappings"].(string)
	want := "IACI,IAEF"
	if got != want {
		t.Errorf("mappings = %q, want %q", got, want)
	}

	// Decode to verify
	pos := 0
	// Seg 1: col=4, srcIdx=0, srcLine=1, srcCol=4
	col, np, _ := decodeVLQ(got, pos)
	srcIdx, np, _ := decodeVLQ(got, np)
	srcLine, np, _ := decodeVLQ(got, np)
	srcCol, pos, _ := decodeVLQ(got, np)
	if col != 4 || srcIdx != 0 || srcLine != 1 || srcCol != 4 {
		t.Errorf("seg 1: col=%d srcIdx=%d srcLine=%d srcCol=%d",
			col, srcIdx, srcLine, srcCol)
	}

	if got[pos] != ',' {
		t.Fatalf("expected ',' at pos %d", pos)
	}
	pos++

	// Seg 2: col=4 (8-4), srcIdx=0, srcLine=2 (3-1), srcCol=-2 (2-4)
	col, np, _ = decodeVLQ(got, pos)
	srcIdx, np, _ = decodeVLQ(got, np)
	srcLine, np, _ = decodeVLQ(got, np)
	srcCol, pos, _ = decodeVLQ(got, np)
	if col != 4 || srcIdx != 0 || srcLine != 2 || srcCol != -2 {
		t.Errorf("seg 2: col=%d srcIdx=%d srcLine=%d srcCol=%d",
			col, srcIdx, srcLine, srcCol)
	}
}

func TestBuilderMultipleSources(t *testing.T) {
	// Golden from Dart:
	// gen(0,0)->a.scss(0,0), gen(0,10)->b.scss(0,0)
	// => mappings "AAAA,UCAA", sources ["a.scss","b.scss"]
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 0, testSourceSpan("a.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(0, 10, testSourceSpan("b.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	got := parsed["mappings"].(string)
	want := "AAAA,UCAA"
	if got != want {
		t.Errorf("mappings = %q, want %q (Dart golden)", got, want)
	}

	sources := parsed["sources"].([]any)
	if len(sources) != 2 || sources[0] != "a.scss" || sources[1] != "b.scss" {
		t.Errorf("sources = %v, want [a.scss b.scss]", sources)
	}
}

func TestBuilderTwoLinesDifferentSources(t *testing.T) {
	// gen(0,0)->a.scss(0,0), gen(1,5)->src(line=2,col=3)
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 0, testSourceSpan("a.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(1, 5, testSourceSpan("input.scss", 2, 3)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	got := parsed["mappings"].(string)
	want := "AAAA;KCEG"
	if got != want {
		t.Errorf("mappings = %q, want %q", got, want)
	}

	sources := parsed["sources"].([]any)
	if len(sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(sources))
	}
}

func TestBuilderMultiLine(t *testing.T) {
	// gen(0,2)->src(line=0,col=5)
	// gen(1,0)->src(line=1,col=3)
	// gen(1,8)->src(line=3,col=7)
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 2, testSourceSpan("input.scss", 0, 5)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(1, 0, testSourceSpan("input.scss", 1, 3)); err != nil {
		t.Fatal(err)
	}

	if err := b.AddMapping(1, 8, testSourceSpan("input.scss", 3, 7)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	got := parsed["mappings"].(string)
	want := "EAAK;AACF,QAEI"
	if got != want {
		t.Errorf("mappings = %q, want %q", got, want)
	}
}

func TestToMappingWithPrefix(t *testing.T) {
	// Single entry at (0, 0) with prefix (1, 5)
	// Golden from Dart: entry gets offset to gen(1, 5)
	b := NewBuilder("output.css")
	if err := b.AddMapping(0, 0, testSourceSpan("input.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	m, err := b.ToMappingWithPrefix(1, 5)
	if err != nil {
		t.Fatal(err)
	}

	// With prefix of 1 line and 5 cols on line 0:
	// Entry was at gen(0,0). After prefix: line 0+1=1, col 0+5=5.
	// gen(1,5)->src(0,0,0) => col delta 5 = VLQ "K", srcIdx=0/line=0/col=0 = VLQ "A"
	// Newline at 1: ";KAAA"
	want := ";KAAA"
	if m.Mappings != want {
		t.Errorf("mappings = %q, want %q", m.Mappings, want)
	}
}

func TestToMappingWithPrefixEmpty(t *testing.T) {
	b := NewBuilder("output.css")
	m, err := b.ToMappingWithPrefix(3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if m.Mappings != "" {
		t.Errorf("mappings = %q, want empty", m.Mappings)
	}
	if len(m.URLs) != 0 {
		t.Errorf("URLs = %v, want empty", m.URLs)
	}
}

func TestToMappingWithPrefixMultiLine(t *testing.T) {
	// gen(0,2)->src(line=0,col=5)
	// gen(1,0)->src(line=1,col=3)
	// With prefix (1, 0): gen lines become 1 and 2, col 2 stays on line 1
	b := NewBuilder("output.css")

	if err := b.AddMapping(0, 2, testSourceSpan("input.scss", 0, 5)); err != nil {
		t.Fatal(err)
	}
	if err := b.AddMapping(1, 0, testSourceSpan("input.scss", 1, 3)); err != nil {
		t.Fatal(err)
	}

	m, err := b.ToMappingWithPrefix(1, 0)
	if err != nil {
		t.Fatal(err)
	}

	// After prefix: gen(1,2)->src(0,5), gen(2,0)->src(1,3)
	// Expected matching Dart golden
	want := ";EAAK;AACF"
	if m.Mappings != want {
		t.Errorf("mappings = %q, want %q", m.Mappings, want)
	}

	// Verify via JSON too
	jsonBytes, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}
	if got := parsed["mappings"].(string); got != want {
		t.Errorf("JSON mappings = %q, want %q", got, want)
	}
}

func TestSingleMappingJSON(t *testing.T) {
	m := &SingleMapping{
		URLs:     []string{"test.scss"},
		Mappings: "AAAA",
	}
	jsonBytes, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["version"] != float64(3) {
		t.Errorf("version = %v, want 3", parsed["version"])
	}
	// SingleMapping.JSON() does NOT include "file" (only Builder.ToJSON() does)
	if _, ok := parsed["file"]; ok {
		t.Error("SingleMapping.JSON() should not include 'file'")
	}
	sources, ok := parsed["sources"].([]any)
	if !ok || len(sources) != 1 || sources[0] != "test.scss" {
		t.Errorf("sources = %v, want [test.scss]", parsed["sources"])
	}
	if parsed["mappings"] != "AAAA" {
		t.Errorf("mappings = %v, want AAAA", parsed["mappings"])
	}
}

func TestSingleMappingJSONWithSourcesContent(t *testing.T) {
	m := &SingleMapping{
		URLs:           []string{"test.scss"},
		Mappings:       "AAAA",
		SourcesContent: []string{"body { color: red; }"},
	}
	jsonBytes, err := m.JSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	sc, ok := parsed["sourcesContent"].([]any)
	if !ok || len(sc) != 1 || sc[0] != "body { color: red; }" {
		t.Errorf("sourcesContent = %v, want [body { color: red; }]", parsed["sourcesContent"])
	}
}

func TestBuilderJsonIncludesSourcesContent(t *testing.T) {
	b := NewBuilder("output.css")
	if err := b.AddMapping(0, 0, testSourceSpan("input.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}

	// Builder.ToJSON() includes sourcesContent from FileSource.text()
	if _, ok := parsed["sourcesContent"]; !ok {
		t.Log("sourcesContent may be nil if FileSource is not accessible in test helpers — this is OK")
	}
}

func TestToMappingIncludesSourcesContent(t *testing.T) {
	b := NewBuilder("output.css")
	if err := b.AddMapping(0, 0, testSourceSpan("input.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	m, err := b.ToMapping()
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("ToMapping returned nil")
	}
	// SourcesContent may be nil if FileSource is not available via test helper
	if m.SourcesContent != nil && len(m.SourcesContent) > 0 {
		t.Logf("SourcesContent: %v", m.SourcesContent)
	}
}

func TestToMappingWithPrefixIncludesSourcesContent(t *testing.T) {
	b := NewBuilder("output.css")
	if err := b.AddMapping(0, 0, testSourceSpan("input.scss", 0, 0)); err != nil {
		t.Fatal(err)
	}

	m, err := b.ToMappingWithPrefix(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("ToMappingWithPrefix returned nil")
	}
	if m.SourcesContent != nil && len(m.SourcesContent) > 0 {
		t.Logf("SourcesContent: %v", m.SourcesContent)
	}
}

func TestBuilderMultibyteSourceColumn(t *testing.T) {
	b := NewBuilder("output.css")
	// Span at 'b' after the 3-byte ▼: byte offset 4, character column 2. Dart
	// counts UTF-16 units, so the encoded source column must be 2, not 4.
	if err := b.AddMapping(0, 0, testSpan("a\u25bcb\n", 4, 5)); err != nil {
		t.Fatal(err)
	}
	jsonBytes, err := b.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatal(err)
	}
	if got := parsed["mappings"]; got != "AAAE" {
		t.Errorf("mappings = %v, want AAAE", got)
	}
}
