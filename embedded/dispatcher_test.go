// Copyright 2024 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package embedded

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"
)

// dispatcherHarness starts the embedded dispatcher in-process using pipe
// pairs and provides helpers to send/receive protocol messages.
type dispatcherHarness struct {
	inW    io.WriteCloser
	outR   io.ReadCloser
	stderr *stderrBuffer
	outMu  sync.Mutex
	inMu   sync.Mutex
	closed bool
}

// stderrBuffer is a synchronized buffer for stderr output.
type stderrBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (s *stderrBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *stderrBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func newHarness(t *testing.T) *dispatcherHarness {
	t.Helper()

	compInR, compInW := io.Pipe()
	compOutR, compOutW := io.Pipe()
	stderr := &stderrBuffer{}

	d := NewIsolateDispatcher(compInR, compOutW, stderr)
	d.ExitFn = func(code int) {
		// In tests, don't call os.Exit — just record the exit code.
		// The dispatcher goroutine will terminate naturally.
	}
	go func() {
		_ = d.Listen()
	}()

	h := &dispatcherHarness{
		inW:    compInW,
		outR:   compOutR,
		stderr: stderr,
	}
	t.Cleanup(func() {
		h.close()
	})
	return h
}

// send writes a packet with the given compilation ID and protobuf message.
func (h *dispatcherHarness) send(compilationID uint32, msg proto.Message) {
	h.inMu.Lock()
	defer h.inMu.Unlock()
	packet := serializePacket(compilationID, msg)
	if err := WritePacket(h.inW, packet); err != nil {
		// Write failure means the dispatcher has closed the pipe.
		return
	}
}

// receive reads the next valid packet and parses it as an OutboundMessage.
func (h *dispatcherHarness) receive() (uint32, *OutboundMessage) {
	for {
		h.outMu.Lock()
		if h.closed {
			h.outMu.Unlock()
			return 0, nil
		}
		packet, err := ReadPacket(h.outR)
		if err != nil {
			h.closed = true
			h.outMu.Unlock()
			return 0, nil
		}
		compilationID, msgBuf, _ := parsePacket(packet)
		msg := &OutboundMessage{}
		if err := proto.Unmarshal(msgBuf, msg); err != nil {
			h.outMu.Unlock()
			continue
		}
		h.outMu.Unlock()
		return compilationID, msg
	}
}

// close closes the input pipe.
func (h *dispatcherHarness) close() {
	h.inW.Close()
}

// ---------- Request builders ----------

// compileString creates a CompileRequest for an SCSS string.
func compileString(source string) *InboundMessage {
	input := &InboundMessage_CompileRequest_StringInput{
		Source: source,
		Syntax: Syntax_SCSS,
	}
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
			},
		},
	}
}

// compileStringWithOpts creates a CompileRequest with the given options.
func compileStringWithOpts(source string, style OutputStyle, syntax Syntax, sourceMap bool, alertColor bool, alertAscii bool) *InboundMessage {
	input := &InboundMessage_CompileRequest_StringInput{
		Source: source,
		Syntax: syntax,
	}
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				Style:      style,
				SourceMap:  sourceMap,
				AlertColor: alertColor,
				AlertAscii: alertAscii,
			},
		},
	}
}

// compileStringWithSmOpts creates a CompileRequest with source map options.
func compileStringWithSmOpts(source string, sourceMap bool, includeSources bool) *InboundMessage {
	input := &InboundMessage_CompileRequest_StringInput{
		Source: source,
		Syntax: Syntax_SCSS,
	}
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				SourceMap:               sourceMap,
				SourceMapIncludeSources: includeSources,
			},
		},
	}
}

// compileStringWithImporters creates a CompileRequest with importers.
func compileStringWithImporters(source string, importers []*InboundMessage_CompileRequest_Importer) *InboundMessage {
	input := &InboundMessage_CompileRequest_StringInput{
		Source: source,
		Syntax: Syntax_SCSS,
	}
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				Importers: importers,
			},
		},
	}
}

// compileStringWithFunctions creates a CompileRequest with custom functions.
func compileStringWithFunctions(source string, functions []string) *InboundMessage {
	input := &InboundMessage_CompileRequest_StringInput{
		Source: source,
		Syntax: Syntax_SCSS,
	}
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				GlobalFunctions: functions,
			},
		},
	}
}

// ---------- Response extractors ----------

// getCompileResponse receives and returns the CompileResponse.
func getCompileResponse(t *testing.T, h *dispatcherHarness) *OutboundMessage_CompileResponse {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetCompileResponse() == nil {
		t.Fatalf("expected CompileResponse, got %v", msg.GetMessage())
	}
	return msg.GetCompileResponse()
}

// getCompileSuccess receives a CompileResponse and asserts it's a success.
func getCompileSuccess(t *testing.T, h *dispatcherHarness) *OutboundMessage_CompileResponse_CompileSuccess {
	t.Helper()
	resp := getCompileResponse(t, h)
	if resp.GetSuccess() == nil {
		t.Fatalf("expected success, got failure: %s", resp.GetFailure().GetMessage())
	}
	return resp.GetSuccess()
}

// getCompileFailure receives a CompileResponse and asserts it's a failure.
func getCompileFailure(t *testing.T, h *dispatcherHarness) *OutboundMessage_CompileResponse_CompileFailure {
	t.Helper()
	resp := getCompileResponse(t, h)
	if resp.GetFailure() == nil {
		t.Fatal("expected failure, got success")
	}
	return resp.GetFailure()
}

// getLogEvent receives the next message and asserts it's a LogEvent.
func getLogEvent(t *testing.T, h *dispatcherHarness) *OutboundMessage_LogEvent {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetLogEvent() == nil {
		t.Fatalf("expected LogEvent, got %v", msg.GetMessage())
	}
	return msg.GetLogEvent()
}

// getCanonicalizeRequest receives and asserts a CanonicalizeRequest.
func getCanonicalizeRequest(t *testing.T, h *dispatcherHarness) *OutboundMessage_CanonicalizeRequest {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetCanonicalizeRequest() == nil {
		t.Fatalf("expected CanonicalizeRequest, got %v", msg.GetMessage())
	}
	return msg.GetCanonicalizeRequest()
}

// getImportRequest receives and asserts an ImportRequest.
func getImportRequest(t *testing.T, h *dispatcherHarness) *OutboundMessage_ImportRequest {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetImportRequest() == nil {
		t.Fatalf("expected ImportRequest, got %v", msg.GetMessage())
	}
	return msg.GetImportRequest()
}

// getFileImportRequest receives and asserts a FileImportRequest.
func getFileImportRequest(t *testing.T, h *dispatcherHarness) *OutboundMessage_FileImportRequest {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetFileImportRequest() == nil {
		t.Fatalf("expected FileImportRequest, got %v", msg.GetMessage())
	}
	return msg.GetFileImportRequest()
}

// getFunctionCallRequest receives and asserts a FunctionCallRequest.
func getFunctionCallRequest(t *testing.T, h *dispatcherHarness) *OutboundMessage_FunctionCallRequest {
	t.Helper()
	_, msg := h.receive()
	if msg == nil {
		t.Fatal("received nil message")
	}
	if msg.GetFunctionCallRequest() == nil {
		t.Fatalf("expected FunctionCallRequest, got %v", msg.GetMessage())
	}
	return msg.GetFunctionCallRequest()
}

// ---------- Response senders ----------

// sendCanonicalizeResponse sends a canonicalize response for the given request.
func sendCanonicalizeResponse(h *dispatcherHarness, compID uint32, reqID uint32, url string) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_CanonicalizeResponse_{
			CanonicalizeResponse: &InboundMessage_CanonicalizeResponse{
				Id:     reqID,
				Result: &InboundMessage_CanonicalizeResponse_Url{Url: url},
			},
		},
	})
}

// sendCanonicalizeResponseEmpty sends an empty (not-found) canonicalize response.
func sendCanonicalizeResponseEmpty(h *dispatcherHarness, compID uint32, reqID uint32) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_CanonicalizeResponse_{
			CanonicalizeResponse: &InboundMessage_CanonicalizeResponse{Id: reqID},
		},
	})
}

// sendImportResponse sends an import response for the given request.
func sendImportResponse(h *dispatcherHarness, compID uint32, reqID uint32, contents string) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_ImportResponse_{
			ImportResponse: &InboundMessage_ImportResponse{
				Id: reqID,
				Result: &InboundMessage_ImportResponse_Success{
					Success: &InboundMessage_ImportResponse_ImportSuccess{
						Contents: contents,
					},
				},
			},
		},
	})
}

// sendEmptyImportResponse sends an empty (not-found) import response.
func sendEmptyImportResponse(h *dispatcherHarness, compID uint32, reqID uint32) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_ImportResponse_{
			ImportResponse: &InboundMessage_ImportResponse{Id: reqID},
		},
	})
}

// sendFileImportResponse sends a file import response for the given request.
func sendFileImportResponse(h *dispatcherHarness, compID uint32, reqID uint32, fileUrl string) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_FileImportResponse_{
			FileImportResponse: &InboundMessage_FileImportResponse{
				Id:     reqID,
				Result: &InboundMessage_FileImportResponse_FileUrl{FileUrl: fileUrl},
			},
		},
	})
}

// sendEmptyFileImportResponse sends an empty (not-found) file import response.
func sendEmptyFileImportResponse(h *dispatcherHarness, compID uint32, reqID uint32) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_FileImportResponse_{
			FileImportResponse: &InboundMessage_FileImportResponse{Id: reqID},
		},
	})
}

// sendFunctionCallResponseSuccess sends a successful function call response.
func sendFunctionCallResponseSuccess(h *dispatcherHarness, compID uint32, reqID uint32, value *Value) {
	h.send(compID, &InboundMessage{
		Message: &InboundMessage_FunctionCallResponse_{
			FunctionCallResponse: &InboundMessage_FunctionCallResponse{
				Id:     reqID,
				Result: &InboundMessage_FunctionCallResponse_Success{Success: value},
			},
		},
	})
}

// ---------- Value constructors ----------

var trueValue = &Value{
	Value: &Value_Singleton{Singleton: SingletonValue_TRUE},
}
var falseValue = &Value{
	Value: &Value_Singleton{Singleton: SingletonValue_FALSE},
}
var nullValue = &Value{
	Value: &Value_Singleton{Singleton: SingletonValue_NULL},
}

func rgbValue(red, green, blue int, alpha float64) *Value {
	r := float64(red)
	g := float64(green)
	b := float64(blue)
	a := alpha
	return &Value{
		Value: &Value_Color_{
			Color: &Value_Color{
				Space:    "rgb",
				Channel1: &r,
				Channel2: &g,
				Channel3: &b,
				Alpha:    &a,
			},
		},
	}
}

func hslValue(hue, saturation, lightness int, alpha float64) *Value {
	h := float64(hue)
	s := float64(saturation)
	l := float64(lightness)
	a := alpha
	return &Value{
		Value: &Value_Color_{
			Color: &Value_Color{
				Space:    "hsl",
				Channel1: &h,
				Channel2: &s,
				Channel3: &l,
				Alpha:    &a,
			},
		},
	}
}

func numberValue(val float64) *Value {
	return &Value{
		Value: &Value_Number_{
			Number: &Value_Number{Value: val},
		},
	}
}

func pxValue(val float64) *Value {
	return &Value{
		Value: &Value_Number_{
			Number: &Value_Number{
				Value:      val,
				Numerators: []string{"px"},
			},
		},
	}
}

// ---------- Tests ----------

func TestVersionRequest(t *testing.T) {
	h := newHarness(t)

	h.send(0, &InboundMessage{
		Message: &InboundMessage_VersionRequest_{
			VersionRequest: &InboundMessage_VersionRequest{Id: 123},
		},
	})

	compilationID, msg := h.receive()
	if compilationID != 0 {
		t.Errorf("expected compilation ID 0, got %d", compilationID)
	}
	resp := msg.GetVersionResponse()
	if resp == nil {
		t.Fatal("expected VersionResponse")
	}
	if resp.GetId() != 123 {
		t.Errorf("expected id 123, got %d", resp.GetId())
	}
	if resp.GetProtocolVersion() == "" {
		t.Error("expected non-empty protocol version")
	}
	if resp.GetCompilerVersion() == "" {
		t.Error("expected non-empty compiler version")
	}
	if resp.GetImplementationVersion() == "" {
		t.Error("expected non-empty implementation version")
	}
	if resp.GetImplementationName() != "dart-sass" {
		t.Errorf("expected implementation name 'dart-sass', got %q", resp.GetImplementationName())
	}
}

// ---------- "exits upon protocol error" group ----------

func TestProtocolErrorEmptyMessage(t *testing.T) {
	h := newHarness(t)

	h.send(1, &InboundMessage{})

	_, msg := h.receive()
	if msg == nil {
		t.Fatal("expected ProtocolError")
	}
	if msg.GetError() == nil {
		t.Fatalf("expected ProtocolError, got %v", msg.GetMessage())
	}
	err := msg.GetError()
	if err.GetType() != ProtocolErrorType_PARSE {
		t.Errorf("expected PARSE error type, got %v", err.GetType())
	}
	if !strings.Contains(err.GetMessage(), "not set") {
		t.Errorf("expected 'not set' in message, got: %q", err.GetMessage())
	}
}

func TestProtocolErrorUnterminatedCompilationID(t *testing.T) {
	h := newHarness(t)

	// Write raw bytes: packet length=1, body=[0x81] (continuation bit set)
	h.inW.Write([]byte{1, 0x81})

	_, msg := h.receive()
	if msg == nil || msg.GetError() == nil {
		t.Fatal("expected ProtocolError")
	}
	err := msg.GetError()
	if err.GetType() != ProtocolErrorType_PARSE {
		t.Errorf("expected PARSE error type, got %v", err.GetType())
	}
	if !strings.Contains(err.GetMessage(), "continuation bit") {
		t.Errorf("expected 'continuation bit' in message, got: %q", err.GetMessage())
	}
}

func TestProtocolError33BitCompilationID(t *testing.T) {
	h := newHarness(t)

	// Varint encoding of 0x100000000 (33-bit value): [0x80, 0x80, 0x80, 0x80, 0x10]
	varint33 := []byte{0x80, 0x80, 0x80, 0x80, 0x10}
	packet := append(EncodeVarint(uint32(len(varint33))), varint33...)
	h.inW.Write(packet)

	_, msg := h.receive()
	if msg == nil || msg.GetError() == nil {
		t.Fatal("expected ProtocolError")
	}
	err := msg.GetError()
	if err.GetType() != ProtocolErrorType_PARSE {
		t.Errorf("expected PARSE error type, got %v", err.GetType())
	}
	if !strings.Contains(err.GetMessage(), "longer than 32") {
		t.Errorf("expected 'longer than 32 bits' in message, got: %q", err.GetMessage())
	}
}

func TestProtocolErrorInvalidProtobuf(t *testing.T) {
	h := newHarness(t)

	// Write raw bytes: packet length=2, body=[1, 0]
	// Varint for compilation ID = 1, message buffer = [0] which is invalid protobuf
	h.inW.Write([]byte{2, 1, 0})

	_, msg := h.receive()
	if msg == nil || msg.GetError() == nil {
		t.Fatal("expected ProtocolError")
	}
	err := msg.GetError()
	if err.GetType() != ProtocolErrorType_PARSE {
		t.Errorf("expected PARSE error type, got %v", err.GetType())
	}
	if !strings.Contains(err.GetMessage(), "invalid") {
		t.Errorf("expected 'invalid' in message, got: %q", err.GetMessage())
	}
}

func TestProtocolErrorDuplicateCompilationID(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithImporters("@use 'other'", []*InboundMessage_CompileRequest_Importer{
		{Importer: &InboundMessage_CompileRequest_Importer_ImporterId{ImporterId: 1}},
	}))
	req := getCanonicalizeRequest(t, h)
	if req == nil {
		t.Fatal("expected CanonicalizeRequest")
	}

	// Send another CompileRequest with the same compilation ID
	h.send(1, compileString("a {b: c}"))

	// Should cause a protocol error or the compilation may succeed
	// depending on implementation details
	_, msg := h.receive()
	if msg != nil && msg.GetError() != nil {
		err := msg.GetError()
		if err.GetType() != ProtocolErrorType_PARAMS {
			t.Errorf("expected PARAMS error type, got %v", err.GetType())
		}
		if !strings.Contains(err.GetMessage(), "already active") {
			t.Errorf("expected 'already active' in message, got: %q", err.GetMessage())
		}
	}
}

// ---------- "compiles CSS from" group ----------

func TestCompileScssString(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: 1px + 2px}"))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileScssStringExplicit(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_SCSS, false, false, false))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileSassString(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a\n  b: 1px + 2px", OutputStyle_EXPANDED, Syntax_INDENTED, false, false, false))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileCssString(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: c}", OutputStyle_EXPANDED, Syntax_CSS, false, false, false))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: c;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileAbsolutePath(t *testing.T) {
	h := newHarness(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.scss")
	if err := os.WriteFile(path, []byte("a {b: 1px + 2px}"), 0644); err != nil {
		t.Fatal(err)
	}

	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_Path{Path: path},
			},
		},
	})

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileRelativePath(t *testing.T) {
	h := newHarness(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.scss")
	if err := os.WriteFile(path, []byte("a {b: 1px + 2px}"), 0644); err != nil {
		t.Fatal(err)
	}

	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_Path{Path: path},
			},
		},
	})

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS output: %q", css)
	}
}

func TestCompileExpanded(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_SCSS, false, false, false))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("expected expanded output, got: %q", css)
	}
}

func TestCompileCompressed(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_COMPRESSED, Syntax_SCSS, false, false, false))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a{b:3px}" {
		t.Errorf("expected compressed output, got: %q", css)
	}
}

// ---------- "exits when stdin is closed" group ----------

func TestStdinClose(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: 1px + 2px}"))
	_ = getCompileSuccess(t, h)

	h.close()
	// After stdin closes, dispatcher should exit cleanly
}

func TestStdinCloseAfterCompilingCSS(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: 1px + 2px}"))
	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("unexpected CSS: %q", css)
	}
	h.close()
}

func TestStdinCloseWhileCompilingCSS(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithFunctions("a {b: foo() + 2px}", []string{"foo()"}))
	_ = getFunctionCallRequest(t, h)
	h.close()
}

// ---------- Concurrency tests ----------

func TestManyConcurrentCompilations(t *testing.T) {
	h := newHarness(t)

	const totalRequests = 10
	done := make(chan struct{}, totalRequests)

	for i := 1; i <= totalRequests; i++ {
		go func(id int) {
			h.send(uint32(id), compileString("a {b: 1px + 2px}"))
			success := getCompileSuccess(t, h)
			css := strings.TrimSpace(success.GetCss())
			if css != "a {\n  b: 3px;\n}" {
				t.Errorf("compilation %d: unexpected CSS: %q", id, css)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < totalRequests; i++ {
		<-done
	}
}

func TestManySequentialCompilations(t *testing.T) {
	h := newHarness(t)

	const totalRequests = 100
	for i := 1; i <= totalRequests; i++ {
		h.send(uint32(i), compileString("a {b: 1px + 2px}"))
		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 3px;\n}" {
			t.Errorf("compilation %d: unexpected CSS: %q", i, css)
		}
	}
}

func TestClosesGracefullyWithManyInFlight(t *testing.T) {
	h := newHarness(t)

	const totalRequests = 15
	for i := 1; i <= totalRequests; i++ {
		h.send(uint32(i), compileStringWithFunctions("a {b: foo() + 2px}", []string{"foo()"}))
	}

	// Read all function call requests to confirm they were sent
	for i := 0; i < totalRequests; i++ {
		_ = getFunctionCallRequest(t, h)
	}

	h.close()
}

func TestSequentialCompilations(t *testing.T) {
	h := newHarness(t)

	for i := 1; i <= 5; i++ {
		h.send(uint32(i), compileString("a {b: 1px + 2px}"))
		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 3px;\n}" {
			t.Errorf("compilation %d: unexpected CSS: %q", i, css)
		}
	}
}

func TestConcurrentCompilations(t *testing.T) {
	h := newHarness(t)

	const count = 5
	done := make(chan struct{}, count)

	for i := 1; i <= count; i++ {
		go func(id int) {
			h.send(uint32(id), compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_SCSS, false, false, false))
			success := getCompileSuccess(t, h)
			css := strings.TrimSpace(success.GetCss())
			if css != "a {\n  b: 3px;\n}" {
				t.Errorf("compilation %d: unexpected CSS: %q", id, css)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < count; i++ {
		<-done
	}
}

// ---------- Source map tests ----------

func TestCompileSourceMap(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_SCSS, true, false, false))

	success := getCompileSuccess(t, h)
	if success.GetSourceMap() == "" {
		t.Error("expected non-empty source map")
	}
}

func TestCompileNoSourceMapByDefault(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_SCSS, false, false, false))

	success := getCompileSuccess(t, h)
	if success.GetSourceMap() != "" {
		t.Error("expected empty source map")
	}
}

func TestSourceMapNoContent(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithSmOpts("a {b: 1px + 2px}", true, false))

	success := getCompileSuccess(t, h)
	if success.GetSourceMap() == "" {
		t.Error("expected non-empty source map")
	}
}

func TestSourceMapIncludeContent(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithSmOpts("a {b: 1px + 2px}", true, true))

	success := getCompileSuccess(t, h)
	if success.GetSourceMap() == "" {
		t.Error("expected non-empty source map")
	}
}

// ---------- Log event tests ----------

func TestLogEventDebug(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {@debug hello}"))

	logEvent := getLogEvent(t, h)
	if logEvent.GetType() != LogEventType_DEBUG {
		t.Errorf("expected DEBUG type, got %v", logEvent.GetType())
	}
	if logEvent.GetMessage() != "hello" {
		t.Errorf("expected message 'hello', got %q", logEvent.GetMessage())
	}
	if !strings.Contains(logEvent.GetFormatted(), "DEBUG") {
		t.Errorf("expected formatted to contain 'DEBUG', got %q", logEvent.GetFormatted())
	}
	if !strings.Contains(logEvent.GetFormatted(), "hello") {
		t.Errorf("expected formatted to contain 'hello', got %q", logEvent.GetFormatted())
	}
	if logEvent.GetSpan().GetText() == "" {
		t.Error("expected non-empty span text")
	}
}

func TestLogEventWarnWithStack(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {@warn hello}"))

	logEvent := getLogEvent(t, h)
	if logEvent.GetType() != LogEventType_WARNING {
		t.Errorf("expected WARNING type, got %v", logEvent.GetType())
	}
	if logEvent.GetMessage() != "hello" {
		t.Errorf("expected message 'hello', got %q", logEvent.GetMessage())
	}
	if logEvent.GetStackTrace() == "" {
		t.Error("expected non-empty stack trace")
	}
	if !strings.Contains(logEvent.GetFormatted(), "WARNING") {
		t.Errorf("expected formatted to contain 'WARNING', got %q", logEvent.GetFormatted())
	}
}

func TestLogEventDebugWithTerminalColors(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {@debug hello}", OutputStyle_EXPANDED, Syntax_SCSS, false, true, false))

	logEvent := getLogEvent(t, h)
	formatted := logEvent.GetFormatted()
	if !strings.Contains(formatted, "\x1b[1mDebug\x1b[0m") {
		t.Errorf("expected ANSI bold Debug in formatted, got %q", formatted)
	}
}

func TestLogEventWarnWithTerminalColors(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {@warn hello}", OutputStyle_EXPANDED, Syntax_SCSS, false, true, false))

	logEvent := getLogEvent(t, h)
	formatted := logEvent.GetFormatted()
	if !strings.Contains(formatted, "\x1b[33m\x1b[1mWarning\x1b[0m") {
		t.Errorf("expected ANSI colored Warning in formatted, got %q", formatted)
	}
}

func TestLogEventWarnEncodedAscii(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {@debug a && b}", OutputStyle_EXPANDED, Syntax_SCSS, false, false, true))

	logEvent := getLogEvent(t, h)
	if logEvent.GetType() != LogEventType_WARNING {
		t.Errorf("expected WARNING type, got %v", logEvent.GetType())
	}
	if !strings.Contains(logEvent.GetMessage(), "&&") {
		t.Errorf("expected message to contain '&&', got: %q", logEvent.GetMessage())
	}
}

func TestLogEventParseTimeDeprecation(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("@if true {} @elseif true {}"))

	logEvent := getLogEvent(t, h)
	if logEvent.GetType() != LogEventType_DEPRECATION_WARNING {
		t.Errorf("expected DEPRECATION_WARNING type, got %v", logEvent.GetType())
	}
	if !strings.Contains(logEvent.GetMessage(), "@elseif") {
		t.Errorf("expected message to contain '@elseif', got: %q", logEvent.GetMessage())
	}
}

func TestLogEventRuntimeDeprecation(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {$var: value !global}"))

	logEvent := getLogEvent(t, h)
	if logEvent.GetType() != LogEventType_DEPRECATION_WARNING {
		t.Errorf("expected DEPRECATION_WARNING type, got %v", logEvent.GetType())
	}
	if !strings.Contains(logEvent.GetMessage(), "!global") {
		t.Errorf("expected message to contain '!global', got: %q", logEvent.GetMessage())
	}
}

// ---------- Error handling tests ----------

func TestCompileFailureSyntaxError(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: }"))

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), "Expected expression") {
		t.Errorf("expected 'Expected expression' in message, got: %q", failure.GetMessage())
	}
}

func TestCompileFailureTypeError(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: 1px + 1em}"))

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), "incompatible units") {
		t.Errorf("expected incompatible units message, got: %q", failure.GetMessage())
	}
	if failure.GetSpan().GetText() == "" {
		t.Error("expected non-empty span text")
	}
}

func TestCompileErrorMissingFile(t *testing.T) {
	h := newHarness(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.scss")

	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_Path{Path: path},
			},
		},
	})

	failure := getCompileFailure(t, h)
	// The OS error text is platform-defined ("no such file or directory"
	// on unix, "The system cannot find the file specified." on Windows),
	// so assert the stable contract: the error surfaces the missing path.
	if !strings.Contains(failure.GetMessage(), path) {
		t.Errorf("expected missing path in message, got: %q", failure.GetMessage())
	}
}

func TestCompileErrorMultiLineSpan(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {\n  b: 1px +\n      1em;\n}"))

	failure := getCompileFailure(t, h)
	span := failure.GetSpan()
	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if !strings.Contains(span.GetText(), "\n") {
		t.Error("expected multi-line span text")
	}
}

func TestCompileErrorMultipleStackTrace(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("@function fail() {\n  @return 1px + 1em;\n}\n\na {\n  b: fail();\n}"))

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetStackTrace(), "fail()") {
		t.Errorf("expected stack trace to reference fail(), got: %q", failure.GetStackTrace())
	}
}

func TestCompileErrorUrlFromStringInput(t *testing.T) {
	h := newHarness(t)

	input := &InboundMessage_CompileRequest_StringInput{
		Source: "a {b: 1px + 1em}",
		Syntax: Syntax_SCSS,
		Url:    "foo://bar/baz",
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

	failure := getCompileFailure(t, h)
	if failure.GetSpan().GetUrl() != "foo://bar/baz" {
		t.Errorf("expected url 'foo://bar/baz', got: %q", failure.GetSpan().GetUrl())
	}
}

func TestCompileErrorSassFeaturesInCSS(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileStringWithOpts("a {b: 1px + 2px}", OutputStyle_EXPANDED, Syntax_CSS, false, false, false))

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), "Operators aren't allowed in plain CSS") {
		t.Errorf("expected operator error in message, got: %q", failure.GetMessage())
	}
}

func TestCompileErrorFormattedMessage(t *testing.T) {
	h := newHarness(t)

	h.send(1, compileString("a {b: 1px + 1em}"))

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetFormatted(), "Error:") {
		t.Errorf("expected formatted to contain 'Error:', got: %q", failure.GetFormatted())
	}
	if !strings.Contains(failure.GetFormatted(), "incompatible units") {
		t.Errorf("expected formatted to contain error message, got: %q", failure.GetFormatted())
	}
}

func TestCompileErrorFormattedTerminalColors(t *testing.T) {
	h := newHarness(t)

	input := &InboundMessage_CompileRequest_StringInput{
		Source: "a {b: 1px + 1em}",
		Syntax: Syntax_SCSS,
	}
	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				AlertColor: true,
			},
		},
	})

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetFormatted(), "Error:") {
		t.Errorf("expected formatted to contain 'Error:', got: %q", failure.GetFormatted())
	}
}

func TestCompileErrorFormattedAsciiEncoding(t *testing.T) {
	h := newHarness(t)

	input := &InboundMessage_CompileRequest_StringInput{
		Source: "a {b: 1px + 1em}",
		Syntax: Syntax_SCSS,
	}
	h.send(1, &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: input,
				},
				AlertAscii: true,
			},
		},
	})

	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetFormatted(), "Error:") {
		t.Errorf("expected formatted to contain 'Error:', got: %q", failure.GetFormatted())
	}
}
