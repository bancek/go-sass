// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package embedded

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// canonicalize handles a CanonicalizeRequest and sends a response with a
// canonical URL. Used when testing import requests to avoid duplicating
// canonicalize boilerplate.
func canonicalize(t *testing.T, h *dispatcherHarness) {
	t.Helper()
	request := getCanonicalizeRequest(t, h)
	sendCanonicalizeResponse(h, 1, request.GetId(), "custom:other")
}

func TestImporter_EmitProtocolError(t *testing.T) {
	t.Run("for a response without a corresponding request ID", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		request := getCanonicalizeRequest(t, h)

		// Send response with wrong ID
		h.send(1, &InboundMessage{
			Message: &InboundMessage_CanonicalizeResponse_{
				CanonicalizeResponse: &InboundMessage_CanonicalizeResponse{
					Id: request.GetId() + 1,
				},
			},
		})

		_, msg := h.receive()
		if msg == nil || msg.GetError() == nil {
			t.Fatal("expected ProtocolError")
		}
		err := msg.GetError()
		if !strings.Contains(err.GetMessage(), "doesn't match") {
			t.Errorf("expected 'doesn't match' in message, got: %q", err.GetMessage())
		}
	})

	t.Run("for a response that doesn't match the request type", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		request := getCanonicalizeRequest(t, h)

		// Send wrong response type (ImportResponse instead of CanonicalizeResponse)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_ImportResponse_{
				ImportResponse: &InboundMessage_ImportResponse{Id: request.GetId()},
			},
		})

		_, msg := h.receive()
		if msg == nil || msg.GetError() == nil {
			t.Fatal("expected ProtocolError")
		}
		err := msg.GetError()
		if !strings.Contains(err.GetMessage(), "doesn't match") {
			t.Errorf("expected 'doesn't match' in message, got: %q", err.GetMessage())
		}
	})

	t.Run("for an unset importer", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
			{},
		}))

		_, msg := h.receive()
		if msg == nil || msg.GetError() == nil {
			t.Fatal("expected ProtocolError")
		}
		err := msg.GetError()
		if !strings.Contains(err.GetMessage(), "Missing mandatory") {
			t.Errorf("expected 'Missing mandatory' in message, got: %q", err.GetMessage())
		}
	})

	t.Run("for an importer with nonCanonicalScheme set", func(t *testing.T) {
		t.Run("path", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_Path{Path: "somewhere"}, NonCanonicalScheme: []string{"u"}},
			}))

			_, msg := h.receive()
			if msg == nil || msg.GetError() == nil {
				t.Fatal("expected ProtocolError")
			}
			err := msg.GetError()
			if !strings.Contains(err.GetMessage(), "non_canonical_scheme") {
				t.Errorf("expected 'non_canonical_scheme' in message, got: %q", err.GetMessage())
			}
		})

		t.Run("file importer", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}, NonCanonicalScheme: []string{"u"}},
			}))

			_, msg := h.receive()
			if msg == nil || msg.GetError() == nil {
				t.Fatal("expected ProtocolError")
			}
			err := msg.GetError()
			if !strings.Contains(err.GetMessage(), "non_canonical_scheme") {
				t.Errorf("expected 'non_canonical_scheme' in message, got: %q", err.GetMessage())
			}
		})
	})
}

func TestImporter_Canonicalization(t *testing.T) {
	t.Run("emits a compile failure", func(t *testing.T) {
		startCanonicalize := func(t *testing.T, h *dispatcherHarness) *OutboundMessage_CanonicalizeRequest {
			t.Helper()
			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			}))
			return getCanonicalizeRequest(t, h)
		}

		t.Run("for a canonicalize response with an empty URL", func(t *testing.T) {
			h := newHarness(t)
			request := startCanonicalize(t, h)

			sendCanonicalizeResponse(h, 1, request.GetId(), "")

			failure := getCompileFailure(t, h)
			if !strings.Contains(failure.GetMessage(), "must return an absolute URL") {
				t.Errorf("expected 'must return an absolute URL' in message, got: %q", failure.GetMessage())
			}
			if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
				t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
			}
		})

		t.Run("for a canonicalize response with a relative URL", func(t *testing.T) {
			h := newHarness(t)
			request := startCanonicalize(t, h)

			sendCanonicalizeResponse(h, 1, request.GetId(), "relative")

			failure := getCompileFailure(t, h)
			if !strings.Contains(failure.GetMessage(), "must return an absolute URL") {
				t.Errorf("expected 'must return an absolute URL' in message, got: %q", failure.GetMessage())
			}
		})
	})

	t.Run("includes in CanonicalizeRequest", func(t *testing.T) {
		t.Run("a known importerId", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 5679}},
			}))
			request := getCanonicalizeRequest(t, h)
			if request.GetImporterId() != 5679 {
				t.Errorf("expected importerId 5679, got %d", request.GetImporterId())
			}
		})

		t.Run("the imported URL", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			}))
			request := getCanonicalizeRequest(t, h)
			if request.GetUrl() != "other" {
				t.Errorf("expected URL 'other', got %q", request.GetUrl())
			}
		})
	})

	t.Run("errors cause compilation to fail", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		request := getCanonicalizeRequest(t, h)

		h.send(1, &InboundMessage{
			Message: &InboundMessage_CanonicalizeResponse_{
				CanonicalizeResponse: &InboundMessage_CanonicalizeResponse{
					Id:     request.GetId(),
					Result: &InboundMessage_CanonicalizeResponse_Error{Error: "oh no"},
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "oh no") {
			t.Errorf("expected message to contain 'oh no', got %q", failure.GetMessage())
		}
	})

	t.Run("null results count as not found", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		request := getCanonicalizeRequest(t, h)

		sendCanonicalizeResponseEmpty(h, 1, request.GetId())

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "Can't find stylesheet") {
			t.Errorf("expected \"Can't find stylesheet\" in message, got: %q", failure.GetMessage())
		}
		if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
			t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
		}
	})

	t.Run("the containing URL", func(t *testing.T) {
		t.Run("is unset for a potentially canonical scheme", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters(`@use "u:orange"`, []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			}))
			request := getCanonicalizeRequest(t, h)
			if request.GetContainingUrl() != "" {
				t.Errorf("expected no containingUrl, got %q", request.GetContainingUrl())
			}
		})

		t.Run("for a non-canonical scheme", func(t *testing.T) {
			t.Run("is set to the original URL", func(t *testing.T) {
				h := newHarness(t)

				input := &InboundMessage_CompileRequest_StringInput{
					Source: `@use "u:orange"`,
					Syntax: Syntax_SCSS,
					Url:    "x:original.scss",
				}
				h.send(1, &InboundMessage{
					Message: &InboundMessage_CompileRequest_{
						CompileRequest: &InboundMessage_CompileRequest{
							Input: &InboundMessage_CompileRequest_String_{
								String_: input,
							},
							Importers: []*InboundMessage_CompileRequest_Importer{
								{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{"u"}},
							},
						},
					},
				})
				request := getCanonicalizeRequest(t, h)
				if request.GetContainingUrl() != "x:original.scss" {
					t.Errorf("expected containingUrl 'x:original.scss', got %q", request.GetContainingUrl())
				}
			})

			t.Run("is unset if the original URL is unset", func(t *testing.T) {
				h := newHarness(t)

				h.send(1, compileStringWithImporters(`@use "u:orange"`, []*InboundMessage_CompileRequest_Importer{
					{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{"u"}},
				}))
				request := getCanonicalizeRequest(t, h)
				if request.GetContainingUrl() != "" {
					t.Errorf("expected no containingUrl, got %q", request.GetContainingUrl())
				}
			})
		})

		t.Run("for a schemeless load", func(t *testing.T) {
			t.Run("is set to the original URL", func(t *testing.T) {
				h := newHarness(t)

				input := &InboundMessage_CompileRequest_StringInput{
					Source: `@use "orange"`,
					Syntax: Syntax_SCSS,
					Url:    "x:original.scss",
				}
				h.send(1, &InboundMessage{
					Message: &InboundMessage_CompileRequest_{
						CompileRequest: &InboundMessage_CompileRequest{
							Input: &InboundMessage_CompileRequest_String_{
								String_: input,
							},
							Importers: []*InboundMessage_CompileRequest_Importer{
								{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
							},
						},
					},
				})
				request := getCanonicalizeRequest(t, h)
				if request.GetContainingUrl() != "x:original.scss" {
					t.Errorf("expected containingUrl 'x:original.scss', got %q", request.GetContainingUrl())
				}
			})

			t.Run("is unset if the original URL is unset", func(t *testing.T) {
				h := newHarness(t)

				h.send(1, compileStringWithImporters(`@use "orange"`, []*InboundMessage_CompileRequest_Importer{
					{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
				}))
				request := getCanonicalizeRequest(t, h)
				if request.GetContainingUrl() != "" {
					t.Errorf("expected no containingUrl, got %q", request.GetContainingUrl())
				}
			})
		})
	})

	t.Run("fails if the importer returns a canonical URL with a non-canonical scheme", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{"u"}},
		}))
		request := getCanonicalizeRequest(t, h)

		sendCanonicalizeResponse(h, 1, request.GetId(), "u:other")

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "non-canonical") {
			t.Errorf("expected 'non-canonical' in message, got: %q", failure.GetMessage())
		}
	})

	t.Run("attempts importers in order", func(t *testing.T) {
		h := newHarness(t)

		var importers []*InboundMessage_CompileRequest_Importer
		for i := uint32(0); i < 10; i++ {
			importers = append(importers, &InboundMessage_CompileRequest_Importer{
				Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: i},
			})
		}
		h.send(1, compileStringWithImporters("@use 'other'", importers))

		for i := uint32(0); i < 10; i++ {
			request := getCanonicalizeRequest(t, h)
			if request.GetImporterId() != i {
				t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
			}
			sendCanonicalizeResponseEmpty(h, 1, request.GetId())
		}
	})

	t.Run("tries resolved URL using the original importer first", func(t *testing.T) {
		h := newHarness(t)

		var importers []*InboundMessage_CompileRequest_Importer
		for i := uint32(0); i < 10; i++ {
			importers = append(importers, &InboundMessage_CompileRequest_Importer{
				Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: i},
			})
		}
		h.send(1, compileStringWithImporters("@use 'midstream'", importers))

		// First 5 importers reject (not found)
		for i := uint32(0); i < 5; i++ {
			request := getCanonicalizeRequest(t, h)
			if request.GetUrl() != "midstream" {
				t.Errorf("expected URL 'midstream', got %q", request.GetUrl())
			}
			if request.GetImporterId() != i {
				t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
			}
			sendCanonicalizeResponseEmpty(h, 1, request.GetId())
		}

		// 6th importer returns canonical URL
		canonicalizeReq := getCanonicalizeRequest(t, h)
		if canonicalizeReq.GetImporterId() != 5 {
			t.Errorf("expected importerId 5, got %d", canonicalizeReq.GetImporterId())
		}
		sendCanonicalizeResponse(h, 1, canonicalizeReq.GetId(), "custom:foo/bar")

		// Import request for the canonical URL
		importReq := getImportRequest(t, h)
		sendImportResponse(h, 1, importReq.GetId(), "@use 'upstream'")

		// Next canonicalize should use importer 5 (original importer)
		canonicalizeReq2 := getCanonicalizeRequest(t, h)
		if canonicalizeReq2.GetImporterId() != 5 {
			t.Errorf("expected importerId 5, got %d", canonicalizeReq2.GetImporterId())
		}
		if !strings.Contains(canonicalizeReq2.GetUrl(), "upstream") {
			t.Errorf("expected URL containing 'upstream', got %q", canonicalizeReq2.GetUrl())
		}
	})
}

func TestImporter_Importing(t *testing.T) {
	t.Run("emits a compile failure", func(t *testing.T) {
		t.Run("for an import result with a relative sourceMapUrl", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			}))
			canonicalize(t, h)

			importReq := getImportRequest(t, h)
			srcMapUrl := "relative"
			h.send(1, &InboundMessage{
				Message: &InboundMessage_ImportResponse_{
					ImportResponse: &InboundMessage_ImportResponse{
						Id: importReq.GetId(),
						Result: &InboundMessage_ImportResponse_Success{
							Success: &InboundMessage_ImportResponse_ImportSuccess{
								Contents:     "a {b: c}",
								SourceMapUrl: &srcMapUrl,
							},
						},
					},
				},
			})

			failure := getCompileFailure(t, h)
			if !strings.Contains(failure.GetMessage(), "must return an absolute URL") {
				t.Errorf("expected 'must return an absolute URL' in message, got: %q", failure.GetMessage())
			}
		})
	})

	t.Run("includes in ImportRequest", func(t *testing.T) {
		t.Run("a known importerId", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 5678}},
			}))

			canonicalizeReq := getCanonicalizeRequest(t, h)
			sendCanonicalizeResponse(h, 1, canonicalizeReq.GetId(), "custom:foo")

			importReq := getImportRequest(t, h)
			if importReq.GetImporterId() != 5678 {
				t.Errorf("expected importerId 5678, got %d", importReq.GetImporterId())
			}
		})

		t.Run("the canonical URL", func(t *testing.T) {
			h := newHarness(t)

			h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
				{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			}))

			canonicalizeReq := getCanonicalizeRequest(t, h)
			sendCanonicalizeResponse(h, 1, canonicalizeReq.GetId(), "custom:foo")

			importReq := getImportRequest(t, h)
			if importReq.GetUrl() != "custom:foo" {
				t.Errorf("expected URL 'custom:foo', got %q", importReq.GetUrl())
			}
		})
	})

	t.Run("null results count as not found", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))

		canonicalizeReq := getCanonicalizeRequest(t, h)
		sendCanonicalizeResponse(h, 1, canonicalizeReq.GetId(), "o:other")

		importReq := getImportRequest(t, h)
		sendEmptyImportResponse(h, 1, importReq.GetId())

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "Can't find stylesheet") {
			t.Errorf("expected \"Can't find stylesheet\" in message, got: %q", failure.GetMessage())
		}
		if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
			t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
		}
	})

	t.Run("errors cause compilation to fail", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		canonicalize(t, h)

		importReq := getImportRequest(t, h)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_ImportResponse_{
				ImportResponse: &InboundMessage_ImportResponse{
					Id:     importReq.GetId(),
					Result: &InboundMessage_ImportResponse_Error{Error: "oh no"},
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "oh no") {
			t.Errorf("expected message to contain 'oh no', got %q", failure.GetMessage())
		}
	})

	t.Run("can return an SCSS file", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		canonicalize(t, h)

		importReq := getImportRequest(t, h)
		sendImportResponse(h, 1, importReq.GetId(), "a {b: 1px + 2px}")

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 3px;\n}" {
			t.Errorf("expected 'a { b: 3px; }', got %q", css)
		}
	})

	t.Run("can return an indented syntax file", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		canonicalize(t, h)

		importReq := getImportRequest(t, h)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_ImportResponse_{
				ImportResponse: &InboundMessage_ImportResponse{
					Id: importReq.GetId(),
					Result: &InboundMessage_ImportResponse_Success{
						Success: &InboundMessage_ImportResponse_ImportSuccess{
							Contents: "a\n  b: 1px + 2px",
							Syntax:   Syntax_INDENTED,
						},
					},
				},
			},
		})

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 3px;\n}" {
			t.Errorf("expected 'a { b: 3px; }', got %q", css)
		}
	})

	t.Run("can return a plain CSS file", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))
		canonicalize(t, h)

		importReq := getImportRequest(t, h)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_ImportResponse_{
				ImportResponse: &InboundMessage_ImportResponse{
					Id: importReq.GetId(),
					Result: &InboundMessage_ImportResponse_Success{
						Success: &InboundMessage_ImportResponse_ImportSuccess{
							Contents: "a {b: c}",
							Syntax:   Syntax_CSS,
						},
					},
				},
			},
		})

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: c;\n}" {
			t.Errorf("expected 'a { b: c; }', got %q", css)
		}
	})

	t.Run("uses a data: URL rather than an empty source map URL", func(t *testing.T) {
		h := newHarness(t)
		// sourceMap must be included in the compile request
		srcMapUrl := ""
		t.Log("TODO: source map URL test requires sourceMap:true in compile request")
		_ = h
		_ = srcMapUrl
	})

	t.Run("uses a non-empty source map URL", func(t *testing.T) {
		h := newHarness(t)
		t.Log("TODO: source map URL test requires sourceMap:true in compile request")
		_ = h
	})
}

func TestImporter_HandlesImporterForStringCompile(t *testing.T) {
	h := newHarness(t)

	input := &InboundMessage_CompileRequest_StringInput{
		Source: "@use 'other'",
		Syntax: Syntax_SCSS,
		Importer: &InboundMessage_CompileRequest_Importer{
			Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1},
		},
	}
	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
			},
		},
	})
	canonicalize(t, h)

	importReq := getImportRequest(t, h)
	sendImportResponse(h, 1, importReq.GetId(), "a {b: 1px + 2px}")

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("expected 'a { b: 3px; }', got %q", css)
	}
}

func TestImporter_LoadPaths(t *testing.T) {
	t.Run("are used to load imports", func(t *testing.T) {
		h := newHarness(t)

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "other.scss"), []byte("a {b: c}"), 0644)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_Path{Path: dir}},
		}))
		// The path importer resolves "other" to the file on disk directly,
		// so we get the compile success immediately.
		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: c;\n}" {
			t.Errorf("expected 'a { b: c; }', got %q", css)
		}
	})

	t.Run("are accessed in order", func(t *testing.T) {
		h := newHarness(t)

		var dirs []string
		for i := 0; i < 3; i++ {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, fmt.Sprintf("other%d.scss", i)), []byte(fmt.Sprintf("a {b: %d}", i)), 0644)
			dirs = append(dirs, dir)
		}

		var importers []*InboundMessage_CompileRequest_Importer
		for i := 0; i < 3; i++ {
			importers = append(importers, &InboundMessage_CompileRequest_Importer{
				Importer: &InboundMessage_CompileRequest_Importer_Path{Path: dirs[i]},
			})
		}
		h.send(1, compileStringWithImporters("@use 'other2'", importers))

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 2;\n}" {
			t.Errorf("expected 'a { b: 2; }', got %q", css)
		}
	})

	t.Run("take precedence over later importers", func(t *testing.T) {
		h := newHarness(t)

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "other.scss"), []byte("a {b: c}"), 0644)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_Path{Path: dir}},
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
		}))

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: c;\n}" {
			t.Errorf("expected 'a { b: c; }', got %q", css)
		}
	})

	t.Run("yield precedence to earlier importers", func(t *testing.T) {
		h := newHarness(t)

		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "other.scss"), []byte("a {b: c}"), 0644)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
			{Importer: &InboundMessage_CompileRequest_Importer_Path{Path: dir}},
		}))
		canonicalize(t, h)

		importReq := getImportRequest(t, h)
		sendImportResponse(h, 1, importReq.GetId(), "x {y: z}")

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "x {\n  y: z;\n}" {
			t.Errorf("expected 'x { y: z; }', got %q", css)
		}
	})
}

func TestImporter_FailsCompilationForInvalidScheme(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{""}},
		}))

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "valid URL scheme") {
			t.Errorf("expected 'valid URL scheme' in message, got: %q", failure.GetMessage())
		}
	})

	t.Run("uppercase", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{"U"}},
		}))

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "valid URL scheme") {
			t.Errorf("expected 'valid URL scheme' in message, got: %q", failure.GetMessage())
		}
	})

	t.Run("colon", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("a {b: c}", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}, NonCanonicalScheme: []string{"u:"}},
		}))

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "valid URL scheme") {
			t.Errorf("expected 'valid URL scheme' in message, got: %q", failure.GetMessage())
		}
	})
}
