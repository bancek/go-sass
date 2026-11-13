package embedded

import (
	"strings"
	"testing"
)

// compileStringWithFatalDeprecations builds a CompileRequest with the given
// deprecation selectors.
func compileStringWithFatalDeprecations(source string, fatal, except []string) *InboundMessage {
	return &InboundMessage{
		Message: &InboundMessage_CompileRequest_{
			CompileRequest: &InboundMessage_CompileRequest{
				Input: &InboundMessage_CompileRequest_String_{
					String_: &InboundMessage_CompileRequest_StringInput{
						Source: source,
						Syntax: Syntax_SCSS,
					},
				},
				Silent:                 true,
				FatalDeprecation:       fatal,
				FatalDeprecationExcept: except,
			},
		},
	}
}

// A deprecation excluded through fatal_deprecation_except no longer fails the
// compilation.
//
// Matches Dart: the fatal_deprecation_except loop in _compile
func TestCompileFatalDeprecationExcept(t *testing.T) {
	source := "@import 'other';\na {b: c}\n"

	t.Run("included is fatal", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFatalDeprecations(source, []string{"import"}, nil))
		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), "This is only an error because you've set the import deprecation to be fatal.") {
			t.Errorf("failure = %q", failure.GetMessage())
		}
	})

	t.Run("excluded is no longer fatal", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFatalDeprecations(source, []string{"import"}, []string{"import"}))
		failure := getCompileFailure(t, h)
		if strings.Contains(failure.GetMessage(), "set the import deprecation to be fatal") {
			t.Errorf("failure = %q, want the deprecated @import error", failure.GetMessage())
		}
		if !strings.Contains(failure.GetMessage(), "Can't find stylesheet to import.") {
			t.Errorf("failure = %q, want the import resolution error", failure.GetMessage())
		}
	})
}

// Excluding a deprecation that no selector included is a protocol error.
//
// Matches Dart: the paramsError thrown by _compile
func TestCompileFatalDeprecationExceptNotIncluded(t *testing.T) {
	h := newHarness(t)
	h.send(1, compileStringWithFatalDeprecations("a {b: c}", nil, []string{"import"}))

	_, msg := h.receive()
	protocolErr := msg.GetError()
	if protocolErr == nil {
		t.Fatalf("expected a protocol error, got %T", msg.GetMessage())
	}
	if protocolErr.GetType() != ProtocolErrorType_PARAMS {
		t.Errorf("protocol error type = %v, want PARAMS", protocolErr.GetType())
	}
	if protocolErr.GetMessage() != `The deprecation "import" was excluded from a DeprecationSelectorDifference that didn't include it.` {
		t.Errorf("protocol error message = %q", protocolErr.GetMessage())
	}
}
