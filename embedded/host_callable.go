// Copyright 2019 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package embedded

// dart-source: lib/src/embedded/host_callable.dart

import (
	"fmt"

	"github.com/bancek/go-sass/evalcontext"
	"github.com/bancek/go-sass/functions"
	"github.com/bancek/go-sass/sasscallable"
	"github.com/bancek/go-sass/value"
)

// NewHostCallable returns a Callable that invokes a function defined on the
// host with the given signature.
//
// When id is non-nil the host function is addressed by that opaque ID, which
// is how anonymous host functions (with no stable name) are reached;
// otherwise the call goes out under the name parsed from the signature. Each
// invocation protofies its arguments through a fresh per-call Protofier,
// sends a FunctionCallRequest, and deprotofies a success response; a host
// error result fails the call and a missing result reports a mandatory-field
// protocol error. An invalid signature returns an error, where Dart throws a
// SassException. Dart additionally forwards a ProtocolError through the
// dispatcher's error channel; Go returns it to the evaluator's error path,
// which owns reporting for the compilation.
//
// Matches Dart: hostCallable in host_callable.dart.
func NewHostCallable(
	dispatcher *CompilationDispatcher,
	fns *OpaqueRegistry,
	mixins *OpaqueRegistry,
	signature string,
	id *uint32,
) (sasscallable.Callable, error) {
	var callable sasscallable.Callable
	var err error
	callable, err = functions.NewCallableFromSignature(signature,
		func(ec *evalcontext.EvaluationContext, args []value.Value) (value.Value, error) {
			protofier := NewProtofier(dispatcher, fns, mixins)
			request := &OutboundMessage_FunctionCallRequest{}
			for _, arg := range args {
				protoVal, err := protofier.Protofy(arg)
				if err != nil {
					return nil, err
				}
				request.Arguments = append(request.Arguments, protoVal)
			}

			if id != nil {
				request.Identifier = &OutboundMessage_FunctionCallRequest_FunctionId{FunctionId: *id}
			} else {
				request.Identifier = &OutboundMessage_FunctionCallRequest_Name{Name: callable.Name()}
			}

			response, err := dispatcher.sendFunctionCallRequest(request)
			if err != nil {
				return nil, err
			}

			switch response.GetResult().(type) {
			case *InboundMessage_FunctionCallResponse_Success:
				return protofier.DeprotofyResponse(response)

			case *InboundMessage_FunctionCallResponse_Error:
				return nil, fmt.Errorf("%s", response.GetError())

			default:
				return nil, mandatoryError("FunctionCallResponse.result")
			}
		}, true)
	return callable, err
}
