// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/utils.dart

import (
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/proto"

	"github.com/bancek/go-sass/eval"
	"github.com/bancek/go-sass/sasscommon"
)

// errorId is the special ID that indicates an error that's not associated with
// a specific inbound request ID.
//
// The dispatcher overwrites it with the real request ID when one is
// available; it only survives on errors that arrive outside any request.
//
// Matches Dart: const errorId = 0xffffffff in utils.dart.
const errorId uint32 = 0xffffffff

// mandatoryError returns a ProtocolError indicating that a mandatory field
// with the given fieldName was missing.
//
// It reports the absence as a PARAMS error so the host can tell a malformed
// message apart from a parse failure.
//
// Matches Dart: mandatoryError in utils.dart.
func mandatoryError(fieldName string) *ProtocolError {
	return paramsError("Missing mandatory field " + fieldName)
}

// paramsError returns a ProtocolError indicating that the parameters for an
// inbound message were invalid.
//
// The ID defaults to errorId here; the dispatcher replaces it with the
// request ID when one is available.
//
// Matches Dart: paramsError in utils.dart.
func paramsError(message string) *ProtocolError {
	return &ProtocolError{
		Type:    ProtocolErrorType_PARAMS,
		Id:      errorId,
		Message: message,
	}
}

// parseError returns a ProtocolError with type PARSE and the given message.
//
// Unlike paramsError it carries no request ID: a malformed compilation ID or
// varint cannot be attributed to any inbound request.
//
// Matches Dart: parseError in utils.dart.
func parseError(message string) *ProtocolError {
	return &ProtocolError{
		Type:    ProtocolErrorType_PARSE,
		Message: message,
	}
}

// protofySpan converts a Sass source span to a protocol buffer source span.
//
// A nil span (Go has no non-nullable span type, unlike Dart's parameter)
// yields an empty span rather than failing, so logging paths can always send
// a well-formed LogEvent. Otherwise text, start/end offsets, and the source
// URL are copied across; an absent URL leaves the proto URL empty, matching
// Dart's `sourceUrl?.toString() ?? ""`. The span context that Dart forwards
// for SourceSpanWithContext has no proto field here and is intentionally
// dropped.
//
// Matches Dart: protofySpan in utils.dart.
func protofySpan(span *sasscommon.FileSpan) *SourceSpan {
	if span == nil || *span == nil {
		return &SourceSpan{
			Start: &SourceSpan_SourceLocation{},
			End:   &SourceSpan_SourceLocation{},
		}
	}
	s := *span
	start, _ := s.StartLocation()
	end, _ := s.EndLocation()
	text, _ := s.SpanText()
	sourceURL, _ := s.SourceURL()

	protoSpan := &SourceSpan{
		Text: text,
		Start: &SourceSpan_SourceLocation{
			Offset: uint32(start.Offset),
			Line:   uint32(start.Line),
			Column: uint32(start.Column),
		},
		End: &SourceSpan_SourceLocation{
			Offset: uint32(end.Offset),
			Line:   uint32(end.Line),
			Column: uint32(end.Column),
		},
	}

	if sourceURL != nil {
		protoSpan.Url = sourceURL.String()
	}
	return protoSpan
}

// syntaxToSyntax converts a protocol buffer syntax enum into a Go Syntax
// enum. It returns an error for unknown syntax values, where Dart throws a
// string because its switch is exhaustive over the known protos.
//
// Matches Dart: syntaxToSyntax in utils.dart.
func syntaxToSyntax(syntax Syntax) (eval.Syntax, error) {
	switch syntax {
	case Syntax_SCSS:
		return eval.SyntaxSCSS, nil
	case Syntax_INDENTED:
		return eval.SyntaxSass, nil
	case Syntax_CSS:
		return eval.SyntaxCSS, nil
	default:
		return 0, fmt.Errorf("Unknown syntax %v", syntax)
	}
}

// serializeVarint encodes value as an unsigned varint.
//
// It delegates to EncodeVarint, which carries the bit-packing rationale
// (7 payload bits per byte, high bit marks continuation).
//
// Matches Dart: serializeVarint in utils.dart.
func serializeVarint(value uint32) []byte {
	return EncodeVarint(value)
}

// serializePacket encodes a compilation ID and protobuf message into a
// packet buffer as specified by the embedded protocol: the varint-encoded
// ID followed by the marshalled message bytes.
//
// Dart writes through a CodedBufferWriter; Go marshals with proto.Marshal
// first and concatenates, which is equivalent for a single message.
//
// Matches Dart: serializePacket in utils.dart.
func serializePacket(compilationID uint32, message proto.Message) []byte {
	varint := serializeVarint(compilationID)
	msgBytes, _ := proto.Marshal(message)

	packet := make([]byte, len(varint)+len(msgBytes))
	copy(packet, varint)
	copy(packet[len(varint):], msgBytes)
	return packet
}

// parsePacket parses a compilation ID and encoded protobuf message from a
// packet buffer as specified by the embedded protocol.
//
// Bytes feed a VarintBuilder until it completes; the remainder of the buffer
// is the message payload. Dart reuses a shared builder guarded by
// try/finally, while Go allocates a per-call builder (with a deferred Reset)
// so concurrent compilations cannot race on the builder state. A truncated
// packet whose continuation bit never clears reports "continuation bit
// always set", and an over-long ID is wrapped as a PARSE protocol error.
//
// Matches Dart: parsePacket in utils.dart.
func parsePacket(packet []byte) (compilationID uint32, messageBuffer []byte, err error) {
	builder := NewVarintBuilder(32, "compilation ID")
	defer builder.Reset()

	for i := range packet {
		id, done, buildErr := builder.Add(packet[i])
		if buildErr != nil {
			return 0, nil, parseError("Invalid compilation ID: " + buildErr.Error())
		}
		if done {
			return uint32(id), packet[i+1:], nil
		}
	}
	return 0, nil, parseError("Invalid compilation ID: continuation bit always set.")
}

// handleError wraps an error object into a ProtocolError, writes the error to
// stderrWriter (or os.Stderr if nil), and returns the ProtocolError.
//
// A ProtocolError means the host caused the failure: it is stamped with the
// request ID (or errorId when there is none), reported as "Host caused <type>
// error", and sets exit code 76 (EX_PROTOCOL). Any other error is an internal
// compiler bug: the message plus stack trace is printed as "Internal compiler
// error" and wrapped as an INTERNAL error with exit code 70 (EX_SOFTWARE).
// Dart reads the stack from a StackTrace parameter and the ambient stderr;
// Go takes both as explicit parameters so the dispatcher stays testable.
//
// Matches Dart: handleError in utils.dart.
func handleError(err error, stack string, messageID *uint32, stderrWriter io.Writer) *ProtocolError {
	stderrOut := stderrWriter
	if stderrOut == nil {
		stderrOut = os.Stderr
	}
	if pe, ok := AsProtocolError(err); ok {
		id := errorId
		if messageID != nil {
			id = *messageID
		}
		pe.Id = id
		typeName := pe.GetType().String()
		_, _ = fmt.Fprintf(stderrOut, "Host caused %s error", typeName)
		if id != errorId {
			_, _ = fmt.Fprintf(stderrOut, " with request %d", id)
		}
		_, _ = fmt.Fprintf(stderrOut, ": %s\n", pe.GetMessage())
		exitCode = 76 // EX_PROTOCOL
		return pe
	}

	errMsg := err.Error()
	if stack != "" {
		errMsg = errMsg + "\n" + stack
	}
	_, _ = fmt.Fprintf(stderrOut, "Internal compiler error: %s", errMsg)
	exitCode = 70 // EX_SOFTWARE
	return &ProtocolError{
		Type:    ProtocolErrorType_INTERNAL,
		Id:      messageIDOr(messageID, errorId),
		Message: errMsg,
	}
}

// messageIDOr returns messageID if non-nil, otherwise returns fallback.
//
// Small helper so handleError can stamp errorId onto request-less errors
// without repeating the nil check at each construction site.
func messageIDOr(messageID *uint32, fallback uint32) uint32 {
	if messageID != nil {
		return *messageID
	}
	return fallback
}

// AsProtocolError checks whether err is a *ProtocolError and returns it.
//
// Go-only extraction for the type test Dart inlines in handleError; the
// boolean distinguishes a host-caused protocol failure from an internal one.
func AsProtocolError(err error) (*ProtocolError, bool) {
	pe, ok := err.(*ProtocolError)
	return pe, ok
}

// exitCode tracks the process exit code for the embedded compiler, mirroring
// Dart's top-level exitCode variable: 64 for usage errors, 70 for internal
// compiler errors, 76 for host-caused protocol errors. A failed compilation
// itself is a normal failure response and leaves the code untouched.
var exitCode int
