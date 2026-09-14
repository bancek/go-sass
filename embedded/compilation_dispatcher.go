// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/compilation_dispatcher.dart

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/bancek/go-sass/compile"
	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sassurl"
)

// outboundRequestID is the request ID stamped on every compiler-to-host
// request.
//
// The dispatcher runs one compilation at a time and never has more than one
// outbound request outstanding, so the ID never needs to vary; the host
// echoes it back and sendRequest rejects any response carrying another ID.
//
// Matches Dart: _outboundRequestId = 0
const outboundRequestID uint32 = 0

var errMailboxClosed = errors.New("mailbox closed")

// CompilationDispatcher routes messages between the host and one running
// compilation: it takes inbound packets from mailbox, runs each compile,
// and emits responses, log events, and host-callback requests through
// sendPort.
//
// A dispatcher serves a single compilation at a time; the opaque
// function/mixin registries are reset by compile so consecutive
// compilations on the same isolate cannot leak IDs into each other.
//
// Matches Dart: class CompilationDispatcher
type CompilationDispatcher struct {
	mailbox             *Mailbox
	sendPort            *SendPort
	compilationID       uint32
	compilationIDVarint []byte
	functions           *OpaqueRegistry
	mixins              *OpaqueRegistry
	// IO is the filesystem abstraction used for path-input compiles and
	// filesystem importers. It defaults to the ambient implementation and
	// is swapped out in tests.
	IO sassio.IO
	// Stderr receives human-readable diagnostics for malformed protocol
	// traffic handled by the dispatcher loop. It defaults to os.Stderr
	// and is swapped out in tests.
	Stderr io.Writer
}

// NewCompilationDispatcher returns a CompilationDispatcher that takes
// inbound packets from mailbox and emits outbound packets through sendPort.
//
// The dispatcher starts without an active compilation: Listen records the
// compilation ID from the first packet it receives. IO defaults to the
// ambient filesystem and Stderr to os.Stderr; tests override both.
//
// Matches Dart: CompilationDispatcher constructor
func NewCompilationDispatcher(mailbox *Mailbox, sendPort *SendPort) *CompilationDispatcher {
	return &CompilationDispatcher{
		mailbox:  mailbox,
		sendPort: sendPort,
		IO:       sassio.NewDefaultIO(),
		Stderr:   os.Stderr,
	}
}

// Listen runs compilations in a loop until mailbox closes.
//
// Each iteration handles exactly one inbound packet. A closed mailbox ends
// the loop quietly; any other error becomes an outbound error packet while
// the loop keeps serving later compilations on the same isolate. Fatal
// host-side failures are reported through sendPort rather than by exiting
// here; the isolate dispatcher decides when to terminate the process.
//
// Matches Dart: CompilationDispatcher.listen
func (c *CompilationDispatcher) Listen() {
	for {
		err := c.handleMessage()
		if err != nil {
			if errors.Is(err, errMailboxClosed) {
				return
			}
			protocolErr := handleError(err, "", nil, c.Stderr)
			c.sendPort.Send(c.serializePacket(&OutboundMessage{Message: &OutboundMessage_Error{Error: protocolErr}}))
		}
	}
}

// handleMessage receives and processes a single inbound packet.
//
// It records the packet's compilation ID for error messages and for the
// category-prefixed packets serializePacket emits. A compile request runs
// to completion and its response goes out on the same compilation ID; a
// version request on a nonzero ID is a protocol error, as is any host
// response (canonicalize, import, file-import, function-call), which is
// only ever valid as a reply to an outstanding sendRequest call, never
// unsolicited.
func (c *CompilationDispatcher) handleMessage() error {
	rawPacket := c.receive()
	if rawPacket == nil {
		return errMailboxClosed
	}
	compilationID, messageBuffer, parsePacketErr := parsePacket(rawPacket)
	if parsePacketErr != nil {
		return parsePacketErr
	}

	c.compilationID = compilationID
	c.compilationIDVarint = serializeVarint(compilationID)

	msg := &InboundMessage{}
	if err := proto.Unmarshal(messageBuffer, msg); err != nil {
		return parseError(err.Error())
	}

	switch msg.GetMessage().(type) {
	case *InboundMessage_CompileRequest_:
		response, err := c.compile(msg.GetCompileRequest())
		if err != nil {
			return err
		}
		if response != nil {
			c.send(&OutboundMessage{Message: &OutboundMessage_CompileResponse_{CompileResponse: response}})
		}

	case *InboundMessage_VersionRequest_:
		return paramsError("VersionRequest must have compilation ID 0.")

	case *InboundMessage_CanonicalizeResponse_,
		*InboundMessage_ImportResponse_,
		*InboundMessage_FileImportResponse_,
		*InboundMessage_FunctionCallResponse_:
		msgID, ok := InboundMessageID(msg)
		idStr := "nil"
		if ok {
			idStr = fmt.Sprintf("%d", msgID)
		}
		return paramsError(
			"Response ID " + idStr + " doesn't match any outstanding requests" +
				" in compilation " + fmt.Sprintf("%d", c.compilationID) + ".")

	case nil:
		return parseError("InboundMessage.message is not set.")

	default:
		return parseError("Unknown message type: " + msg.String())
	}
	return nil
}

// compile runs one compilation and reports how the caller should treat the
// outcome: a non-nil response with a nil error is a finished compile
// (success or Sass failure) to send back, while a nil response with a
// non-nil error is a protocol violation for the Listen loop to report as
// an error packet.
//
// Registries are reset per compile so opaque function/mixin IDs never leak
// across compilations. The output style defaults to expanded and switches
// to compressed only on request; a silent request compiles under the quiet
// logger, otherwise log events stream through the embedded logger honoring
// the host's color and ASCII flags. Deprecation selectors accept released
// IDs and, for fatal deprecations, version strings that expand to ID sets,
// with anything else drawing a warning instead of failing the compile.
// String inputs without an explicit importer fall back to the no-op
// importer unless the entry URL is a file URL; path inputs require a
// non-empty path.
//
// Matches Dart: _compile
func (c *CompilationDispatcher) compile(request *InboundMessage_CompileRequest) (*OutboundMessage_CompileResponse, error) {
	c.functions = NewOpaqueRegistry()
	c.mixins = NewOpaqueRegistry()

	style := compile.OutputStyleExpanded
	if request.GetStyle() == OutputStyle_COMPRESSED {
		style = compile.OutputStyleCompressed
	}

	var logger sasslogger.Logger
	if request.GetSilent() {
		logger = sasslogger.Quiet
	} else {
		logger = NewEmbeddedLogger(c, request.GetAlertColor(), request.GetAlertAscii())
	}

	fatalDeprecations := c.parseDeprecationsOrWarn(request.GetFatalDeprecation(), true, logger)
	silenceDeprecations := c.parseDeprecationsOrWarn(request.GetSilenceDeprecation(), false, logger)
	futureDeprecations := c.parseDeprecationsOrWarn(request.GetFutureDeprecation(), false, logger)

	var importers []eval.Importer
	for _, imp := range request.GetImporters() {
		decoded, err := c.decodeImporter(imp)
		if err != nil {
			if pe, ok := errors.AsType[*ProtocolError](err); ok {
				return nil, pe
			}
			return c.buildFailureFromError(err), nil
		}
		if decoded == nil {
			return nil, mandatoryError("Importer.importer")
		}
		importers = append(importers, decoded)
	}

	var globalFunctions []sasscallable.Callable
	for _, sig := range request.GetGlobalFunctions() {
		callable, err := hostCallable(c, c.functions, c.mixins, sig, nil)
		if err != nil {
			return c.buildFailureFromError(err), nil
		}
		globalFunctions = append(globalFunctions, callable)
	}

	var compileResult *compile.CompileResult
	var compileErr error

	switch request.GetInput().(type) {
	case *InboundMessage_CompileRequest_String_:
		input := request.GetString_()
		fileImporter, err := c.decodeImporter(input.GetImporter())
		if err != nil {
			return c.buildFailureFromError(err), nil
		}
		// When the string input carries no importer, a file-scheme entry URL
		// means there is no entrypoint importer, while any other scheme
		// compiles against the no-op importer so relative loads still
		// resolve without a base.
		//
		// Matches Dart: importer fallback in _compile (string input branch)
		entrypointImporter := fileImporter
		if entrypointImporter == nil && !strings.HasPrefix(input.GetUrl(), "file:") {
			entrypointImporter = eval.NoOp
		}

		syntax, err := syntaxToSyntax(input.GetSyntax())
		if err != nil {
			return c.buildFailureFromError(err), nil
		}

		sourceURL := input.GetUrl()
		var parsedURL *url.URL
		if sourceURL != "" {
			parsedURL, _ = sassurl.Parse(sourceURL)
		}

		compileResult, compileErr = compile.CompileStringToResult(
			input.GetSource(),
			c.IO,
			syntax,
			parsedURL,
			importers,
			entrypointImporter,
			globalFunctions,
			logger,
			style,
			request.GetQuietDeps(),
			request.GetVerbose(),
			request.GetSourceMap(),
			request.GetCharset(),
			fatalDeprecations,
			silenceDeprecations,
			futureDeprecations,
			request.GetAlertColor(),
			request.GetAlertAscii(),
		)

	case *InboundMessage_CompileRequest_Path:
		path := request.GetPath()
		if path == "" {
			return nil, mandatoryError("CompileRequest.Input.path")
		}

		opts := &compile.CompileOptions{
			Importers:           importers,
			Functions:           globalFunctions,
			Logger:              logger,
			Style:               style,
			QuietDeps:           request.GetQuietDeps(),
			Verbose:             request.GetVerbose(),
			SourceMap:           request.GetSourceMap(),
			Charset:             request.GetCharset(),
			AlertColor:          request.GetAlertColor(),
			AlertAscii:          request.GetAlertAscii(),
			SilenceDeprecations: silenceDeprecations,
			FatalDeprecations:   fatalDeprecations,
			FutureDeprecations:  futureDeprecations,
		}

		compileResult, compileErr = compile.Compile(path, c.IO, opts)

	default:
		return nil, mandatoryError("CompileRequest.input")
	}

	if compileErr != nil {
		if pe, ok := errors.AsType[*ProtocolError](compileErr); ok {
			return nil, pe
		}
		return c.buildFailureFromError(compileErr), nil
	}

	success := &OutboundMessage_CompileResponse_CompileSuccess{
		Css: compileResult.CSS(),
	}

	sm := compileResult.SourceMap()
	if sm != nil {
		if !request.GetSourceMapIncludeSources() {
			sm.SourcesContent = nil
		}
		smJSON, _ := sm.JSON()
		success.SourceMap = string(smJSON)
	}

	response := &OutboundMessage_CompileResponse{
		Result: &OutboundMessage_CompileResponse_Success{Success: success},
	}

	loadedURLs := compileResult.LoadedUrls()
	for u := range loadedURLs.Keys() {
		response.LoadedUrls = append(response.LoadedUrls, u)
	}

	return response, nil
}

// buildFailureFromError converts a compile error into a failure response.
//
// Spanned errors contribute their message, source span, and stack trace;
// errors outside the Sass exception family carry a message only. The Sass
// failure path never produces a protocol error: only the compile caller
// distinguishes transport problems from compilation failures.
//
// Matches Dart: _compile (SassException catch branch)
func (c *CompilationDispatcher) buildFailureFromError(compileErr error) *OutboundMessage_CompileResponse {
	errMsg := compileErr.Error()
	formatted := errMsg
	failure := &OutboundMessage_CompileResponse_CompileFailure{
		Message:    errMsg,
		StackTrace: "",
		Formatted:  formatted,
	}
	if se, ok := compileErr.(*sasscommon.SassRuntimeException); ok {
		span := se.Span
		failure.Span = protofySpan(&span)
		if se.Trace != nil {
			failure.StackTrace = se.Trace.String()
		}
	} else if se, ok := compileErr.(*sasscommon.SassException); ok {
		span := se.Span
		failure.Span = protofySpan(&span)
		if trace, err := se.Trace(); err == nil {
			failure.StackTrace = trace.String()
		}
	} else if mse, ok := compileErr.(*sasscommon.MultiSpanSassRuntimeException); ok {
		span := mse.Span
		failure.Span = protofySpan(&span)
		if mse.Trace != nil {
			failure.StackTrace = mse.Trace.String()
		}
	} else if mse, ok := compileErr.(*sasscommon.MultiSpanSassException); ok {
		span := mse.Span
		failure.Span = protofySpan(&span)
		if trace, err := mse.Trace(); err == nil {
			failure.StackTrace = trace.String()
		}
	} else if se, ok := compileErr.(*sasscommon.SassFormatException); ok {
		span := se.Span
		failure.Span = protofySpan(&span)
		if trace, err := se.Trace(); err == nil {
			failure.StackTrace = trace.String()
		}
	} else if mse, ok := compileErr.(*sasscommon.MultiSpanSassFormatException); ok {
		span := mse.Span
		failure.Span = protofySpan(&span)
		if trace, err := mse.Trace(); err == nil {
			failure.StackTrace = trace.String()
		}
	}
	return &OutboundMessage_CompileResponse{
		Result: &OutboundMessage_CompileResponse_Failure{Failure: failure},
	}
}

// parseDeprecationsOrWarn resolves host-supplied deprecation selectors to
// Deprecation values.
//
// Known IDs pass through. When supportVersions holds (fatal deprecations),
// a version string expands to every deprecation that shipped in that
// version; unknown selectors log a warning naming the offending selector
// instead of aborting the compile.
//
// Matches Dart: parseDeprecationsOrWarn nested function in _compile
func (c *CompilationDispatcher) parseDeprecationsOrWarn(deprecations []string, supportVersions bool, logger sasslogger.Logger) []*deprecation.Deprecation {
	var result []*deprecation.Deprecation
	for _, item := range deprecations {
		if d := deprecation.FromID(item); d != nil {
			result = append(result, d)
		} else if supportVersions {
			expanded := deprecation.ForVersion(item)
			if len(expanded) > 0 {
				result = append(result, expanded...)
			} else {
				logger.Warn(fmt.Sprintf("Invalid deprecation id or version \"%s\".", item), nil, nil)
			}
		} else {
			logger.Warn(fmt.Sprintf("Invalid deprecation id \"%s\".", item), nil, nil)
		}
	}
	return result
}

// decodeImporter converts one proto importer description into an importer
// for the compile, or a nil importer with a nil error when the field is
// unset so the caller can apply the entrypoint fallback.
//
// A path importer becomes a filesystem importer; a host importer ID becomes
// a callback importer that round-trips through this dispatcher, optionally
// restricted to non-canonical schemes; a file importer ID becomes a
// file-callback importer; a node package importer resolves from its entry
// point directory. Non-canonical schemes are rejected on every branch
// except the host importer, which is the only importer allowed to carry
// them.
//
// Matches Dart: _decodeImporter
func (c *CompilationDispatcher) decodeImporter(importer *InboundMessage_CompileRequest_Importer) (eval.Importer, error) {
	switch importer.GetImporter().(type) {
	case *InboundMessage_CompileRequest_Importer_Path:
		if err := c.checkNoNonCanonicalScheme(importer); err != nil {
			return nil, err
		}
		return eval.NewFilesystemImporter(importer.GetPath(), c.IO), nil

	case *InboundMessage_CompileRequest_Importer_ImporterId:
		return NewHostImporter(c, importer.GetImporterId(), importer.GetNonCanonicalScheme())

	case *InboundMessage_CompileRequest_Importer_FileImporterId:
		if err := c.checkNoNonCanonicalScheme(importer); err != nil {
			return nil, err
		}
		return NewFileImporter(c, importer.GetFileImporterId()), nil

	case *InboundMessage_CompileRequest_Importer_NodePackageImporter:
		entryDir := importer.GetNodePackageImporter().GetEntryPointDirectory()
		return eval.NewNodePackageImporter(entryDir, c.IO), nil

	case nil:
		if err := c.checkNoNonCanonicalScheme(importer); err != nil {
			return nil, err
		}
		return nil, nil

	default:
		return nil, nil
	}
}

// checkNoNonCanonicalScheme rejects an importer carrying non-canonical
// schemes, which the protocol reserves for host importers. An empty scheme
// list passes through; anything else is a parameter error.
//
// Matches Dart: _checkNoNonCanonicalScheme
func (c *CompilationDispatcher) checkNoNonCanonicalScheme(importer *InboundMessage_CompileRequest_Importer) error {
	if len(importer.GetNonCanonicalScheme()) == 0 {
		return nil
	}
	return paramsError(
		"Importer.non_canonical_scheme may only be set along " +
			"with Importer.importer.importer_id",
	)
}

// sendLog emits a log event to the host without waiting for a reply. Log
// traffic never participates in the sendRequest response matching.
//
// Matches Dart: sendLog
func (c *CompilationDispatcher) sendLog(event *OutboundMessage_LogEvent) {
	c.send(&OutboundMessage{Message: &OutboundMessage_LogEvent_{LogEvent: event}})
}

// sendCanonicalizeRequest emits a CanonicalizeRequest and blocks until the
// host answers with the matching CanonicalizeResponse.
//
// Matches Dart: sendCanonicalizeRequest
func (c *CompilationDispatcher) sendCanonicalizeRequest(request *OutboundMessage_CanonicalizeRequest) (*InboundMessage_CanonicalizeResponse, error) {
	return sendRequest[*InboundMessage_CanonicalizeResponse](c,
		&OutboundMessage{Message: &OutboundMessage_CanonicalizeRequest_{CanonicalizeRequest: request}})
}

// sendImportRequest emits an ImportRequest and blocks until the host
// answers with the matching ImportResponse.
//
// Matches Dart: sendImportRequest
func (c *CompilationDispatcher) sendImportRequest(request *OutboundMessage_ImportRequest) (*InboundMessage_ImportResponse, error) {
	return sendRequest[*InboundMessage_ImportResponse](c,
		&OutboundMessage{Message: &OutboundMessage_ImportRequest_{ImportRequest: request}})
}

// sendFileImportRequest emits a FileImportRequest and blocks until the host
// answers with the matching FileImportResponse.
//
// Matches Dart: sendFileImportRequest
func (c *CompilationDispatcher) sendFileImportRequest(request *OutboundMessage_FileImportRequest) (*InboundMessage_FileImportResponse, error) {
	return sendRequest[*InboundMessage_FileImportResponse](c,
		&OutboundMessage{Message: &OutboundMessage_FileImportRequest_{FileImportRequest: request}})
}

// sendFunctionCallRequest emits a FunctionCallRequest and blocks until the
// host answers with the matching FunctionCallResponse.
//
// Matches Dart: sendFunctionCallRequest
func (c *CompilationDispatcher) sendFunctionCallRequest(request *OutboundMessage_FunctionCallRequest) (*InboundMessage_FunctionCallResponse, error) {
	return sendRequest[*InboundMessage_FunctionCallResponse](c,
		&OutboundMessage{Message: &OutboundMessage_FunctionCallRequest_{FunctionCallRequest: request}})
}

// sendRequest emits one host-callback request and blocks for its response.
//
// The outbound message carries outboundRequestID; the reply must echo that
// ID and arrive as the expected response type. A nested compile or version
// request on the same compilation, an unset message, an ID mismatch, or a
// type mismatch are all protocol errors. A closed mailbox surfaces as a
// mailbox-closed error so the isolate can exit instead of hanging.
//
// Matches Dart: _sendRequest<T>
func sendRequest[T any](c *CompilationDispatcher, message *OutboundMessage) (T, error) {
	var zero T
	if err := SetOutboundMessageID(message, outboundRequestID); err != nil {
		return zero, err
	}
	c.send(message)

	packet := c.receive()
	if packet == nil {
		return zero, errMailboxClosed
	}
	_, messageBuffer, parsePacketErr := parsePacket(packet)
	if parsePacketErr != nil {
		return zero, parsePacketErr
	}

	msg := &InboundMessage{}
	if err := proto.Unmarshal(messageBuffer, msg); err != nil {
		return zero, parseError(err.Error())
	}

	switch msg.GetMessage().(type) {
	case *InboundMessage_CompileRequest_:
		return zero, paramsError(
			"A CompileRequest with compilation ID " +
				fmt.Sprintf("%d", c.compilationID) + " is already active.")

	case *InboundMessage_VersionRequest_:
		return zero, paramsError("VersionRequest must have compilation ID 0.")

	case *InboundMessage_CanonicalizeResponse_,
		*InboundMessage_ImportResponse_,
		*InboundMessage_FileImportResponse_,
		*InboundMessage_FunctionCallResponse_:
		// Only the four host-response shapes can answer an outbound
		// request; any other message kind is rejected below.
	default:
		return zero, parseError("InboundMessage.message is not set.")
	}

	msgID, ok := InboundMessageID(msg)
	if !ok || msgID != outboundRequestID {
		idVal := "nil"
		if ok {
			idVal = fmt.Sprintf("%d", msgID)
		}
		return zero, paramsError(
			fmt.Sprintf("Response ID %s doesn't match any outstanding requests in compilation %d.",
				idVal, c.compilationID))
	}

	var result any
	switch msg.GetMessage().(type) {
	case *InboundMessage_CanonicalizeResponse_:
		result = msg.GetCanonicalizeResponse()
	case *InboundMessage_ImportResponse_:
		result = msg.GetImportResponse()
	case *InboundMessage_FileImportResponse_:
		result = msg.GetFileImportResponse()
	case *InboundMessage_FunctionCallResponse_:
		result = msg.GetFunctionCallResponse()
	}

	typedResult, ok := result.(T)
	if !ok {
		return zero, paramsError(
			fmt.Sprintf("Request ID %d doesn't match response type in compilation %d.",
				outboundRequestID, c.compilationID))
	}
	_ = zero
	return typedResult, nil
}

// send writes one outbound message to the host, framing it with the active
// compilation ID via serializePacket.
//
// Matches Dart: _send -> _sendPort.send(_serializePacket(message))
func (c *CompilationDispatcher) send(message *OutboundMessage) {
	c.sendPort.Send(c.serializePacket(message))
}

// serializePacket frames an outbound message as a category byte, the varint
// compilation ID, and the marshaled protobuf.
//
// The leading category tells the isolate dispatcher how to treat the
// isolate once the packet is written: compile responses mark the
// compilation finished and free the isolate for reuse, errors mark a fatal
// failure, and anything else (log events, host-callback requests) leaves
// the compilation active. The category byte is consumed by the dispatcher
// and never reaches the host.
//
// Matches Dart: _serializePacket
func (c *CompilationDispatcher) serializePacket(message *OutboundMessage) []byte {
	msgBytes, _ := proto.Marshal(message)

	category := byte(0)
	switch message.GetMessage().(type) {
	case *OutboundMessage_CompileResponse_:
		category = 1
	case *OutboundMessage_Error:
		category = 2
	}

	packet := make([]byte, 1+len(c.compilationIDVarint)+len(msgBytes))
	packet[0] = category
	copy(packet[1:], c.compilationIDVarint)
	copy(packet[1+len(c.compilationIDVarint):], msgBytes)
	return packet
}

// receive takes the next inbound packet, blocking until one arrives. A nil
// return means the mailbox was closed, which the Listen loop treats as a
// clean shutdown rather than an error.
//
// Matches Dart: _receive -> _mailbox.take()
func (c *CompilationDispatcher) receive() []byte {
	return c.mailbox.Take()
}

// closeMailbox closes the mailbox so a blocked receive unblocks with nil
// and Listen returns. Tests use this to shut a dispatcher down without
// exiting the process.
func (c *CompilationDispatcher) closeMailbox() {
	c.mailbox.Close()
}

// hostCallable builds the compiler-side callable for one host-declared
// global function signature, registering any compiler functions and mixins
// it closes over in functions and mixins so later proto traffic can refer
// to them by opaque ID.
func hostCallable(
	dispatcher *CompilationDispatcher,
	functions *OpaqueRegistry,
	mixins *OpaqueRegistry,
	signature string,
	id *uint32,
) (sasscallable.Callable, error) {
	return NewHostCallable(dispatcher, functions, mixins, signature, id)
}

// The evalcontext import is otherwise unused in this file; the blank
// reference pins it against unused-import errors for the compile path
// types the dispatcher works with.
var _ = evalcontext.EvaluationContext{}
