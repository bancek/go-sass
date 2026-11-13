package embedded

import (
	"errors"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// The protofier converts an ArgumentError raised by a value constructor into
// a PARAMS protocol error, matching Dart's `on ArgumentError` handler.
//
// Matches Dart: Protofier._deprotofy
func TestDeprotofyArgumentErrorBecomesParamsProtocolError(t *testing.T) {
	converted := protofierArgumentError(&sasscommon.ArgumentError{
		Message: "If value is null, max must also be null.",
	})
	if converted == nil {
		t.Fatal("ArgumentError was not converted")
	}
	if converted.Type != ProtocolErrorType_PARAMS {
		t.Errorf("protocol error type = %v, want PARAMS", converted.Type)
	}
	if converted.Message != "Invalid argument(s): If value is null, max must also be null." {
		t.Errorf("protocol error message = %q", converted.Message)
	}
}

// A Sass exception is not converted: the dispatcher reports it as a compile
// failure with its own message.
func TestDeprotofySassExceptionIsNotParamsProtocolError(t *testing.T) {
	if converted := protofierArgumentError(sasscommon.NewSassScriptException("boom", nil)); converted != nil {
		t.Errorf("SassScriptException converted to %v, want nil", converted)
	}
}

func TestDeprotofyValidValueHasNoError(t *testing.T) {
	p := NewProtofier(nil, NewOpaqueRegistry(), NewOpaqueRegistry(), NewOpaqueRegistry())

	decoded, err := p.deprotofy(&Value{Value: &Value_Number_{Number: &Value_Number{Value: 1}}})
	if err != nil {
		t.Fatalf("deprotofy: %v", err)
	}
	if number, ok := decoded.(value.SassNumber); !ok || number.NumValue() != 1 {
		t.Fatalf("deprotofy = %v, want the number 1", decoded)
	}
}

// The converted protocol error is what the dispatcher reports to the host.
func TestDeprotofyArgumentErrorReachesDispatcher(t *testing.T) {
	err := protofierArgumentError(&sasscommon.ArgumentError{Message: "bad"})
	if _, ok := errors.AsType[*ProtocolError](err); !ok {
		t.Fatalf("converted error = %T, want *ProtocolError", err)
	}
}
