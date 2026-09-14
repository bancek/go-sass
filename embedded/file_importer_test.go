// Copyright 2021 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package embedded

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fileURLForTest builds the absolute file: URL a host returns for path on
// the current OS: forward slashes with a leading slash (so a Windows path
// becomes file:///C:/..., never the malformed file://C:\... that naive
// "file://"+path concatenation produces).
func fileURLForTest(path string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return "file://" + p
}

func TestFileImporter_EmitProtocolError(t *testing.T) {
	t.Run("for a response without a corresponding request ID", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))
		request := getFileImportRequest(t, h)

		// Send response with wrong ID (request.GetId()+1)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_FileImportResponse_{
				FileImportResponse: &InboundMessage_FileImportResponse{
					Id:     request.GetId() + 1,
					Result: &InboundMessage_FileImportResponse_FileUrl{FileUrl: "file:///test"},
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
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))
		request := getFileImportRequest(t, h)

		// Send wrong response type (CanonicalizeResponse instead of FileImportResponse)
		h.send(1, &InboundMessage{
			Message: &InboundMessage_CanonicalizeResponse_{
				CanonicalizeResponse: &InboundMessage_CanonicalizeResponse{
					Id:     request.GetId(),
					Result: &InboundMessage_CanonicalizeResponse_Url{Url: "custom:foo"},
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
}

func TestFileImporter_EmitCompileFailure(t *testing.T) {
	startFileImport := func(t *testing.T, h *dispatcherHarness) *OutboundMessage_FileImportRequest {
		t.Helper()
		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))
		return getFileImportRequest(t, h)
	}

	t.Run("for a FileImportResponse with a URL that's empty", func(t *testing.T) {
		h := newHarness(t)
		request := startFileImport(t, h)

		h.send(1, &InboundMessage{
			Message: &InboundMessage_FileImportResponse_{
				FileImportResponse: &InboundMessage_FileImportResponse{
					Id:     request.GetId(),
					Result: &InboundMessage_FileImportResponse_FileUrl{FileUrl: ""},
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "must return an absolute URL") {
			t.Errorf("expected 'must return an absolute URL' in message, got: %q", failure.GetMessage())
		}
		if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
			t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
		}
	})

	t.Run("for a FileImportResponse with a URL that's relative", func(t *testing.T) {
		h := newHarness(t)
		request := startFileImport(t, h)

		h.send(1, &InboundMessage{
			Message: &InboundMessage_FileImportResponse_{
				FileImportResponse: &InboundMessage_FileImportResponse{
					Id:     request.GetId(),
					Result: &InboundMessage_FileImportResponse_FileUrl{FileUrl: "foo"},
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "must return an absolute URL") {
			t.Errorf("expected 'must return an absolute URL' in message, got: %q", failure.GetMessage())
		}
	})

	t.Run("for a FileImportResponse with a URL that's not file:", func(t *testing.T) {
		h := newHarness(t)
		request := startFileImport(t, h)

		h.send(1, &InboundMessage{
			Message: &InboundMessage_FileImportResponse_{
				FileImportResponse: &InboundMessage_FileImportResponse{
					Id:     request.GetId(),
					Result: &InboundMessage_FileImportResponse_FileUrl{FileUrl: "other:foo"},
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "must return a file:") {
			t.Errorf("expected 'must return a file:' in message, got: %q", failure.GetMessage())
		}
	})
}

func TestFileImporter_IncludesInRequest(t *testing.T) {
	t.Run("a known importerId", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 5679}},
		}))
		request := getFileImportRequest(t, h)
		if request.GetImporterId() != 5679 {
			t.Errorf("expected importerId 5679, got %d", request.GetImporterId())
		}
	})

	t.Run("the imported URL", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))
		request := getFileImportRequest(t, h)
		if request.GetUrl() != "other" {
			t.Errorf("expected URL 'other', got %q", request.GetUrl())
		}
	})

	t.Run("whether the import came from an @import", func(t *testing.T) {
		h := newHarness(t)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))
		request := getFileImportRequest(t, h)
		if request.GetFromImport() {
			t.Error("expected fromImport to be false for @use")
		}
	})
}

func TestFileImporter_ErrorsCauseCompileFailure(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
		{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
	}))
	request := getFileImportRequest(t, h)

	h.send(1, &InboundMessage{
		Message: &InboundMessage_FileImportResponse_{
			FileImportResponse: &InboundMessage_FileImportResponse{
				Id:     request.GetId(),
				Result: &InboundMessage_FileImportResponse_Error{Error: "oh no"},
			},
		},
	})

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), "oh no") {
		t.Errorf("expected message to contain 'oh no', got %q", failure.GetMessage())
	}
	if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
		t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
	}
}

func TestFileImporter_NullResultsCountAsNotFound(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
		{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
	}))
	request := getFileImportRequest(t, h)

	sendEmptyFileImportResponse(h, 1, request.GetId())

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), "Can't find stylesheet") {
		t.Errorf("expected \"Can't find stylesheet\" in message, got: %q", failure.GetMessage())
	}
	if !strings.Contains(failure.GetSpan().GetText(), "@use 'other'") {
		t.Errorf("expected span text to include @use, got: %q", failure.GetSpan().GetText())
	}
}

func TestFileImporter_AttemptsImportersInOrder(t *testing.T) {
	t.Run("with multiple file importers", func(t *testing.T) {
		h := newHarness(t)

		var importers []*InboundMessage_CompileRequest_Importer
		for i := uint32(0); i < 10; i++ {
			importers = append(importers, &InboundMessage_CompileRequest_Importer{
				Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: i},
			})
		}
		h.send(1, compileStringWithImporters("@use 'other'", importers))

		for i := uint32(0); i < 10; i++ {
			request := getFileImportRequest(t, h)
			if request.GetImporterId() != i {
				t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
			}
			sendEmptyFileImportResponse(h, 1, request.GetId())
		}
	})

	t.Run("with a mixture of file and normal importers", func(t *testing.T) {
		h := newHarness(t)

		var importers []*InboundMessage_CompileRequest_Importer
		for i := uint32(0); i < 10; i++ {
			if i%2 == 0 {
				importers = append(importers, &InboundMessage_CompileRequest_Importer{
					Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: i},
				})
			} else {
				importers = append(importers, &InboundMessage_CompileRequest_Importer{
					Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: i},
				})
			}
		}
		h.send(1, compileStringWithImporters("@use 'other'", importers))

		for i := uint32(0); i < 10; i++ {
			if i%2 == 0 {
				request := getFileImportRequest(t, h)
				if request.GetImporterId() != i {
					t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
				}
				sendEmptyFileImportResponse(h, 1, request.GetId())
			} else {
				request := getCanonicalizeRequest(t, h)
				if request.GetImporterId() != i {
					t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
				}
				sendCanonicalizeResponseEmpty(h, 1, request.GetId())
			}
		}
	})
}

func TestFileImporter_TriesResolvedURLAsRelativePathFirst(t *testing.T) {
	h := newHarness(t)

	dir := t.TempDir()
	midstream := filepath.Join(dir, "midstream.scss")
	upstream := filepath.Join(dir, "upstream.scss")
	os.WriteFile(midstream, []byte("@use 'upstream';"), 0644)
	os.WriteFile(upstream, []byte("a {b: c}"), 0644)

	var importers []*InboundMessage_CompileRequest_Importer
	for i := uint32(0); i < 10; i++ {
		importers = append(importers, &InboundMessage_CompileRequest_Importer{
			Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: i},
		})
	}
	h.send(1, compileStringWithImporters("@use 'midstream'", importers))

	// First 5 file importers reject
	for i := uint32(0); i < 5; i++ {
		request := getFileImportRequest(t, h)
		if request.GetImporterId() != i {
			t.Errorf("expected importerId %d, got %d", i, request.GetImporterId())
		}
		if request.GetUrl() != "midstream" {
			t.Errorf("expected URL 'midstream', got %q", request.GetUrl())
		}
		sendEmptyFileImportResponse(h, 1, request.GetId())
	}

	// 6th file importer returns the file URL
	request := getFileImportRequest(t, h)
	if request.GetImporterId() != 5 {
		t.Errorf("expected importerId 5, got %d", request.GetImporterId())
	}
	sendFileImportResponse(h, 1, request.GetId(), fileURLForTest(midstream))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: c;\n}" {
		t.Errorf("unexpected CSS: %q", css)
	}
}

func TestFileImporter_HandlesStringCompile(t *testing.T) {
	t.Run("without a base URL", func(t *testing.T) {
		h := newHarness(t)
		dir := t.TempDir()
		otherPath := filepath.Join(dir, "other.scss")
		os.WriteFile(otherPath, []byte("a {b: c}"), 0644)

		h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
			{Importer: &InboundMessage_CompileRequest_Importer_FileImporterId{FileImporterId: 1}},
		}))

		request := getFileImportRequest(t, h)
		if request.GetUrl() != "other" {
			t.Errorf("expected URL 'other', got %q", request.GetUrl())
		}

		sendFileImportResponse(h, 1, request.GetId(), fileURLForTest(otherPath))

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: c;\n}" {
			t.Errorf("unexpected CSS: %q", css)
		}
	})
}
