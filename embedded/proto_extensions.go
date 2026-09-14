// Copyright 2023 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

import "fmt"

// dart-source: lib/src/embedded/util/proto_extensions.dart

// InboundMessageID returns the request ID of msg regardless of its concrete
// type, reporting false for message kinds that carry no ID.
//
// Dart models this as a nullable getter on an extension; Go returns an
// explicit ok pair because the language has no extension methods or
// nullable-int returns.
//
// Matches Dart: InboundMessageExtensions.id getter in
// util/proto_extensions.dart.
func InboundMessageID(msg *InboundMessage) (uint32, bool) {
	switch msg.GetMessage().(type) {
	case *InboundMessage_VersionRequest_:
		return msg.GetVersionRequest().GetId(), true
	case *InboundMessage_CanonicalizeResponse_:
		return msg.GetCanonicalizeResponse().GetId(), true
	case *InboundMessage_ImportResponse_:
		return msg.GetImportResponse().GetId(), true
	case *InboundMessage_FileImportResponse_:
		return msg.GetFileImportResponse().GetId(), true
	case *InboundMessage_FunctionCallResponse_:
		return msg.GetFunctionCallResponse().GetId(), true
	default:
		return 0, false
	}
}

// OutboundMessageID returns the ID of msg regardless of its concrete type,
// erroring on message kinds that carry no ID.
//
// Dart throws an ArgumentError there; Go returns an error for the same
// unknown-type case.
//
// Matches Dart: OutboundMessageExtensions.id getter in
// util/proto_extensions.dart.
func OutboundMessageID(msg *OutboundMessage) (uint32, error) {
	switch msg.GetMessage().(type) {
	case *OutboundMessage_CanonicalizeRequest_:
		return msg.GetCanonicalizeRequest().GetId(), nil
	case *OutboundMessage_ImportRequest_:
		return msg.GetImportRequest().GetId(), nil
	case *OutboundMessage_FileImportRequest_:
		return msg.GetFileImportRequest().GetId(), nil
	case *OutboundMessage_FunctionCallRequest_:
		return msg.GetFunctionCallRequest().GetId(), nil
	case *OutboundMessage_VersionResponse_:
		return msg.GetVersionResponse().GetId(), nil
	default:
		return 0, fmt.Errorf("Unknown message type: %s", msg.String())
	}
}

// SetOutboundMessageID sets the ID of msg regardless of its concrete type,
// erroring on message kinds that carry no ID.
//
// It is the free-function form of Dart's extension ID setter, used by the
// dispatcher to stamp the shared outbound request ID onto each request
// before sending.
//
// Matches Dart: OutboundMessageExtensions.id setter in
// util/proto_extensions.dart.
func SetOutboundMessageID(msg *OutboundMessage, id uint32) error {
	switch msg.GetMessage().(type) {
	case *OutboundMessage_CanonicalizeRequest_:
		msg.GetCanonicalizeRequest().Id = id
	case *OutboundMessage_ImportRequest_:
		msg.GetImportRequest().Id = id
	case *OutboundMessage_FileImportRequest_:
		msg.GetFileImportRequest().Id = id
	case *OutboundMessage_FunctionCallRequest_:
		msg.GetFunctionCallRequest().Id = id
	case *OutboundMessage_VersionResponse_:
		msg.GetVersionResponse().Id = id
	default:
		return fmt.Errorf("Unknown message type: %s", msg.String())
	}
	return nil
}
