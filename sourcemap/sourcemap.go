// Copyright 2018 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package sourcemap

// dart-source: lib/src/util/source_map_buffer.dart (lower-level: V3 source map JSON builder)

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/bancek/go-sass/sasscommon"
)

// SingleMapping is a structured version 3 source map: the source URLs,
// the VLQ-encoded mappings string, and the optional per-source contents.
// File feeds the map's "file" field when the mapping is serialized.
//
// Matches Dart: SingleMapping in package:source_maps
type SingleMapping struct {
	// URLs lists the source files used, in the order the mappings index
	// them. Serialized as the map's "sources".
	URLs []string

	// Mappings holds the VLQ-encoded mappings string: generated column,
	// source index, source line, and source column per segment, each
	// relative to the previous value.
	Mappings string

	// SourcesContent holds the optional source text for each source file,
	// indexed in the same order as URLs. Serialized as "sourcesContent"
	// only when source contents are included.
	SourcesContent []string

	// File is the optional target URL emitted as the source map's "file" field.
	// The CLI sets it to the basename of the destination CSS file.
	File string
}

// Builder accumulates source map entries and produces a version 3 source
// map. Entries sort by target position at encode time, since the mappings
// string prints each segment in target order with values relative to the
// previous segment.
//
// Matches Dart: SourceMapBuilder in package:source_maps (builder.dart);
// the entry-dedup and newline bookkeeping live in sourcemapbuffer, which
// feeds this builder.
type Builder struct {
	// file seeds the "file" field of maps encoded directly from the
	// builder.
	file string
	// sources lists the source URLs in first-seen order; entries index
	// into it.
	sources []string
	// sourcesContent parallels sources once consumed; built lazily from
	// sourceFiles at encode time.
	sourcesContent []string
	// sourceFiles remembers one file handle per source index so contents
	// can be emitted when requested.
	sourceFiles map[int]*sasscommon.FileSource
	// entries holds mappings in insertion order; sorted at encode time.
	entries []Entry
}

// Entry maps a position in the generated output to a position in a source
// file. Generated positions count from zero; source lines likewise, with
// source columns in UTF-16 units.
//
// Matches Dart: Entry in package:source_maps (builder.dart)
type Entry struct {
	// GenLine is the zero-based generated line.
	GenLine int
	// GenCol is the zero-based generated column.
	GenCol int
	// SourceIdx indexes the source URL in the enclosing map's sources.
	SourceIdx int
	// SrcLine is the zero-based source line.
	SrcLine int
	// SrcCol is the zero-based source column in UTF-16 units.
	SrcCol int
}

// NewBuilder creates a new source map builder tagging its output with
// the given output file name.
//
// Matches Dart: SourceMapBuilder constructor in package:source_maps
func NewBuilder(file string) *Builder {
	return &Builder{file: file}
}

// AddMapping records a mapping from a generated position to the start of
// span. The source URL joins the sources list on first sight, the file
// handle is remembered for optional contents output, and the source column
// counts UTF-16 units rather than bytes, matching Dart.
//
// Matches Dart: SourceMapBuilder.addLocation/addSpan in package:source_maps
// (fed per span-start by the source map buffer)
func (b *Builder) AddMapping(genLine, genCol int, span sasscommon.FileSpan) error {
	var sourceURL string
	u, err := span.SourceURL()
	if err != nil {
		return err
	}
	if u != nil {
		sourceURL = u.String()
	}
	sourceIdx := -1
	for i, s := range b.sources {
		if s == sourceURL {
			sourceIdx = i
			break
		}
	}
	if sourceIdx == -1 {
		sourceIdx = len(b.sources)
		b.sources = append(b.sources, sourceURL)
	}

	if b.sourceFiles == nil {
		b.sourceFiles = make(map[int]*sasscommon.FileSource)
	}
	if _, ok := b.sourceFiles[sourceIdx]; !ok {
		if file, ferr := span.File(); ferr == nil && file != nil {
			b.sourceFiles[sourceIdx] = file
		}
	}

	startLoc, err := span.StartLocation()
	if err != nil {
		return err
	}

	srcCol := startLoc.Column
	// Dart's source map columns count UTF-16 units, not bytes or runes, so
	// the file's own column conversion replaces the raw span column.
	if file, ferr := span.File(); ferr == nil && file != nil {
		srcCol = file.CharacterColumn(startLoc.Offset)
	}

	b.entries = append(b.entries, Entry{
		GenLine:   genLine,
		GenCol:    genCol,
		SourceIdx: sourceIdx,
		SrcLine:   startLoc.Line,
		SrcCol:    srcCol,
	})
	return nil
}

// LastMapping returns the target (line, column), source line, and source
// column of the last entry, if any.
//
// Matches Rust: sourcemap/mod.rs last_mapping
func (b *Builder) LastMapping() (genLine, genCol, srcLine, srcCol int, ok bool) {
	if len(b.entries) == 0 {
		return 0, 0, 0, 0, false
	}
	e := b.entries[len(b.entries)-1]
	return e.GenLine, e.GenCol, e.SrcLine, e.SrcCol, true
}

// TrimLastIfAt removes the last entry if its target is exactly at
// (genLine, genCol).
//
// Matches Dart: SourceMapBuffer._writeLine's "Trim useless entries" step
// (and Rust: sourcemap/mod.rs trim_last_if_at).
func (b *Builder) TrimLastIfAt(genLine, genCol int) {
	if len(b.entries) > 0 {
		e := b.entries[len(b.entries)-1]
		if e.GenLine == genLine && e.GenCol == genCol {
			b.entries = b.entries[:len(b.entries)-1]
		}
	}
}

// AddMappingLikeLast adds an entry at (genLine, genCol) that reuses the last
// entry's source (used for the in_span continuation across a newline).
//
// Matches Dart: SourceMapBuffer._writeLine's Entry(_entries.last.source,
// _targetLocation) (and Rust: sourcemap/mod.rs add_mapping_like_last).
func (b *Builder) AddMappingLikeLast(genLine, genCol int) {
	if len(b.entries) > 0 {
		e := b.entries[len(b.entries)-1]
		b.entries = append(b.entries, Entry{
			GenLine:   genLine,
			GenCol:    genCol,
			SourceIdx: e.SourceIdx,
			SrcLine:   e.SrcLine,
			SrcCol:    e.SrcCol,
		})
	}
}

// sourceMapJSON is the JSON representation of a version 3 source map.
//
// Field order matches Dart's SingleMapping.toJson (parser.dart:476-488):
// version, sourceRoot, sources, names, mappings, file, sourcesContent.
type sourceMapJSON struct {
	Version        int      `json:"version"`
	SourceRoot     string   `json:"sourceRoot"`
	Sources        []string `json:"sources"`
	Names          []string `json:"names"`
	Mappings       string   `json:"mappings"`
	File           string   `json:"file,omitempty"`
	SourcesContent []string `json:"sourcesContent,omitempty"`
}

// encodeSourceMapJSON marshals a source map without Go's default HTML escaping
// (Dart's jsonEncode leaves <, >, and & untouched), trimming the trailing
// newline that json.Encoder adds.
func encodeSourceMapJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// ToMapping produces a structured SingleMapping from the builder's
// accumulated entries, sorted by target position since the mappings
// string encodes segments in target order.
func (b *Builder) ToMapping() (*SingleMapping, error) {
	if len(b.entries) == 0 {
		return &SingleMapping{
			URLs:           b.sourcesOrEmpty(),
			Mappings:       "",
			SourcesContent: b.buildSourcesContent(),
		}, nil
	}

	sorted := make([]Entry, len(b.entries))
	copy(sorted, b.entries)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].GenLine != sorted[j].GenLine {
			return sorted[i].GenLine < sorted[j].GenLine
		}
		return sorted[i].GenCol < sorted[j].GenCol
	})

	return b.encodeEntries(sorted)
}

// ToMappingWithPrefix produces a SingleMapping from accumulated entries,
// adjusting target positions by prefixLines and prefixCol.
//
// Matches Dart: SourceMapBuffer.buildSourceMap entry adjustment
func (b *Builder) ToMappingWithPrefix(prefixLines, prefixCol int) (*SingleMapping, error) {
	if len(b.entries) == 0 {
		return &SingleMapping{
			URLs:           b.sourcesOrEmpty(),
			Mappings:       "",
			SourcesContent: b.buildSourcesContent(),
		}, nil
	}

	sorted := make([]Entry, len(b.entries))
	copy(sorted, b.entries)
	for i := range sorted {
		if sorted[i].GenLine == 0 {
			sorted[i].GenCol += prefixCol
		}
		sorted[i].GenLine += prefixLines
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].GenLine != sorted[j].GenLine {
			return sorted[i].GenLine < sorted[j].GenLine
		}
		return sorted[i].GenCol < sorted[j].GenCol
	})

	return b.encodeEntries(sorted)
}

// encodeEntries encodes sorted entries into a SingleMapping. Each line
// opens with its segments joined by "," and lines join with ";"; the
// generated column resets per line while source index, line, and column
// stay relative across the whole map.
func (b *Builder) encodeEntries(sorted []Entry) (*SingleMapping, error) {
	var mappings strings.Builder
	var prevGenCol, prevSrcIdx, prevSrcLine, prevSrcCol int
	currentLine := -1
	firstOnLine := true

	for _, e := range sorted {
		for currentLine < e.GenLine {
			if currentLine >= 0 {
				mappings.WriteByte(';')
			}
			currentLine++
			prevGenCol = 0
			firstOnLine = true
		}

		if !firstOnLine {
			mappings.WriteByte(',')
		}
		firstOnLine = false

		v, err := encodeVLQ(e.GenCol - prevGenCol)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevGenCol = e.GenCol

		v, err = encodeVLQ(e.SourceIdx - prevSrcIdx)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcIdx = e.SourceIdx

		v, err = encodeVLQ(e.SrcLine - prevSrcLine)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcLine = e.SrcLine

		v, err = encodeVLQ(e.SrcCol - prevSrcCol)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcCol = e.SrcCol
	}

	return &SingleMapping{
		URLs:           b.sourcesOrEmpty(),
		Mappings:       mappings.String(),
		SourcesContent: b.buildSourcesContent(),
	}, nil
}

// JSON produces the source map as JSON bytes.
//
// The "sourcesContent" field is included when non-empty, matching the
// historical behavior of this method. Use [SingleMapping.JSONWithTarget] to
// control source contents explicitly (as the CLI does via --embed-sources).
func (m *SingleMapping) JSON() ([]byte, error) {
	return m.JSONWithTarget(m.File, len(m.SourcesContent) > 0)
}

// JSONWithTarget produces the source map as JSON bytes, using file as the
// "file" field and including sourcesContent when includeSourceContents is set.
//
// Matches Dart: SingleMapping.toJson(includeSourceContents:)
func (m *SingleMapping) JSONWithTarget(file string, includeSourceContents bool) ([]byte, error) {
	out := sourceMapJSON{
		Version:    3,
		SourceRoot: "",
		Sources:    m.URLs,
		Names:      []string{},
		Mappings:   m.Mappings,
		File:       file,
	}
	if out.Sources == nil {
		out.Sources = []string{}
	}
	if includeSourceContents {
		out.SourcesContent = m.SourcesContent
		if out.SourcesContent == nil {
			out.SourcesContent = []string{}
		}
	}
	return encodeSourceMapJSON(out)
}

// ToJSON produces the source map as JSON bytes, tagging it with the
// builder's file name and including contents for sources seen so far.
func (b *Builder) ToJSON() ([]byte, error) {
	if len(b.entries) == 0 {
		return encodeSourceMapJSON(sourceMapJSON{
			Version:    3,
			SourceRoot: "",
			File:       b.file,
			Sources:    b.sourcesOrEmpty(),
			Names:      []string{},
			Mappings:   "",
		})
	}

	sorted := make([]Entry, len(b.entries))
	copy(sorted, b.entries)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].GenLine != sorted[j].GenLine {
			return sorted[i].GenLine < sorted[j].GenLine
		}
		return sorted[i].GenCol < sorted[j].GenCol
	})

	var mappings strings.Builder
	var prevGenCol, prevSrcIdx, prevSrcLine, prevSrcCol int
	currentLine := -1
	firstOnLine := true

	for _, e := range sorted {
		for currentLine < e.GenLine {
			if currentLine >= 0 {
				mappings.WriteByte(';')
			}
			currentLine++
			prevGenCol = 0
			firstOnLine = true
		}

		if !firstOnLine {
			mappings.WriteByte(',')
		}
		firstOnLine = false

		v, err := encodeVLQ(e.GenCol - prevGenCol)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevGenCol = e.GenCol

		v, err = encodeVLQ(e.SourceIdx - prevSrcIdx)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcIdx = e.SourceIdx

		v, err = encodeVLQ(e.SrcLine - prevSrcLine)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcLine = e.SrcLine

		v, err = encodeVLQ(e.SrcCol - prevSrcCol)
		if err != nil {
			return nil, err
		}
		mappings.WriteString(v)
		prevSrcCol = e.SrcCol
	}

	result := sourceMapJSON{
		Version:        3,
		SourceRoot:     "",
		File:           b.file,
		Sources:        b.sourcesOrEmpty(),
		SourcesContent: b.buildSourcesContent(),
		Names:          []string{},
		Mappings:       mappings.String(),
	}

	return encodeSourceMapJSON(result)
}

// buildSourcesContent collects one contents string per source URL, in
// sources order, leaving unseen sources empty. Nil when nothing was
// recorded, so callers can omit the field.
func (b *Builder) buildSourcesContent() []string {
	if b.sourceFiles == nil || len(b.sources) == 0 {
		return nil
	}
	result := make([]string, len(b.sources))
	for i := range b.sources {
		if file, ok := b.sourceFiles[i]; ok {
			result[i] = file.Text()
		}
	}
	return result
}

// sourcesOrEmpty returns the recorded source URLs, or an empty list when
// none were recorded, so encoded maps always carry a "sources" array.
func (b *Builder) sourcesOrEmpty() []string {
	if b.sources == nil {
		return []string{}
	}
	return b.sources
}
