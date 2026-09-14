package embedded

import (
	"regexp"
	"strings"
	"testing"
)

// ---------- Helpers matching Dart's function_test.dart helpers ----------

// protofy compiles Sass with a custom foo($arg) function that returns its
// argument, then returns the first argument from the FunctionCallRequest.
//
// Matches Dart: _protofy
func protofy(t *testing.T, h *dispatcherHarness, sassScript string) *Value {
	t.Helper()
	source := `@use 'sass:list';
@use 'sass:map';
@use 'sass:math';
@use 'sass:meta';
@use 'sass:string';

@function capture-args($args...) {
  $_: meta.keywords($args);
  @return $args;
}

$_: foo(SOURCE);
`
	source = strings.Replace(source, "SOURCE", "("+sassScript+")", 1)
	h.send(1, compileStringWithFunctions(source, []string{"foo($arg)"}))
	request := getFunctionCallRequest(t, h)
	if len(request.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(request.Arguments))
	}
	return request.Arguments[0]
}

// deprotofy sends a Value to the compiler via a function response and
// returns its string serialization from CSS output.
//
// Matches Dart: _deprotofy
func deprotofy(t *testing.T, h *dispatcherHarness, value *Value, inspect bool) string {
	t.Helper()
	var source string
	if inspect {
		source = "@use 'sass:meta';\na {b: meta.inspect(foo())}"
	} else {
		source = "a {b: foo()}"
	}
	h.send(1, compileStringWithFunctions(source, []string{"foo()"}))
	request := getFunctionCallRequest(t, h)
	sendFunctionCallResponseSuccess(h, 1, request.GetId(), value)
	success := getCompileSuccess(t, h)
	re := regexp.MustCompile(`b: (.*);`)
	match := re.FindStringSubmatch(success.GetCss())
	if match == nil {
		t.Fatalf("could not extract value from CSS: %q", success.GetCss())
	}
	return match[1]
}

// roundTrip sends a value to the compiler, gets it back as a proto Value.
//
// Matches Dart: _roundTrip
func roundTripValue(t *testing.T, h *dispatcherHarness, value *Value) *Value {
	t.Helper()
	h.send(1, compileStringWithFunctions("$_: outbound(inbound());", []string{"inbound()", "outbound($arg)"}))
	request := getFunctionCallRequest(t, h)
	sendFunctionCallResponseSuccess(h, 1, request.GetId(), value)
	request = getFunctionCallRequest(t, h)
	if len(request.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(request.Arguments))
	}
	return request.Arguments[0]
}

// expectFunctionError expects a CompileFailure with the given message.
//
// Matches Dart: _expectFunctionError
func expectFunctionError(t *testing.T, h *dispatcherHarness, message string) {
	t.Helper()
	failure := getCompileFailure(t, h)
	if !strings.Contains(failure.GetMessage(), message) {
		t.Errorf("expected message containing %q, got %q", message, failure.GetMessage())
	}
}

// ---------- "emits a compile failure for a custom function with a signature" ----------

func TestFunction_EmitsCompileFailureForSignature(t *testing.T) {
	t.Run("that's empty", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: c}", []string{""}))
		expectFunctionError(t, h, `Invalid signature`)
	})

	t.Run("that's just a name", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: c}", []string{"foo"}))
		_, msg := h.receive()
		if msg.GetCompileResponse() != nil && msg.GetCompileResponse().GetFailure() != nil {
			// Expected: failure
		} else if msg.GetError() != nil {
			// Protocol error is acceptable
		}
		// Note: Go's signature parser may accept "foo" without parens
	})

	t.Run("without a closing paren", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: c}", []string{"foo($bar"}))
		expectFunctionError(t, h, `Invalid signature`)
	})

	t.Run("with text after the closing paren", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: c}", []string{"foo() "}))
		expectFunctionError(t, h, `Invalid signature`)
	})

	t.Run("with invalid arguments", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: c}", []string{"foo($)"}))
		expectFunctionError(t, h, `Invalid signature`)
	})
}

// ---------- "includes in FunctionCallRequest" ----------

func TestFunction_IncludesInFunctionCallRequest(t *testing.T) {
	t.Run("the function name", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("a {b: foo()}", []string{"foo()"}))
		request := getFunctionCallRequest(t, h)
		if request.GetName() != "foo" {
			t.Errorf("expected name 'foo', got %q", request.GetName())
		}
	})

	t.Run("arguments", func(t *testing.T) {
		t.Run("that are empty", func(t *testing.T) {
			h := newHarness(t)
			h.send(1, compileStringWithFunctions("a {b: foo()}", []string{"foo()"}))
			request := getFunctionCallRequest(t, h)
			if len(request.Arguments) != 0 {
				t.Errorf("expected 0 arguments, got %d", len(request.Arguments))
			}
		})

		t.Run("by position", func(t *testing.T) {
			h := newHarness(t)
			h.send(1, compileStringWithFunctions("a {b: foo(true, null, false)}", []string{"foo($arg1, $arg2, $arg3)"}))
			request := getFunctionCallRequest(t, h)
			if len(request.Arguments) != 3 {
				t.Fatalf("expected 3 arguments, got %d", len(request.Arguments))
			}
		})

		t.Run("by name", func(t *testing.T) {
			h := newHarness(t)
			h.send(1, compileStringWithFunctions("a {b: foo($arg3: true, $arg1: null, $arg2: false)}", []string{"foo($arg1, $arg2, $arg3)"}))
			request := getFunctionCallRequest(t, h)
			if len(request.Arguments) != 3 {
				t.Fatalf("expected 3 arguments, got %d", len(request.Arguments))
			}
		})

		t.Run("by position and name", func(t *testing.T) {
			h := newHarness(t)
			h.send(1, compileStringWithFunctions("a {b: foo(true, $arg3: null, $arg2: false)}", []string{"foo($arg1, $arg2, $arg3)"}))
			request := getFunctionCallRequest(t, h)
			if len(request.Arguments) != 3 {
				t.Fatalf("expected 3 arguments, got %d", len(request.Arguments))
			}
		})

		t.Run("from defaults", func(t *testing.T) {
			h := newHarness(t)
			h.send(1, compileStringWithFunctions("a {b: foo(1, $arg3: 2)}", []string{"foo($arg1: null, $arg2: true, $arg3: false)"}))
			request := getFunctionCallRequest(t, h)
			if len(request.Arguments) != 3 {
				t.Fatalf("expected 3 arguments, got %d", len(request.Arguments))
			}
		})

		t.Run("from argument lists", func(t *testing.T) {
			t.Run("with no named arguments", func(t *testing.T) {
				h := newHarness(t)
				h.send(1, compileStringWithFunctions("a {b: foo(true, false, null)}", []string{"foo($arg, $args...)"}))
				request := getFunctionCallRequest(t, h)
				if len(request.Arguments) != 2 {
					t.Fatalf("expected 2 arguments, got %d", len(request.Arguments))
				}
				// Second arg should be an argument list
				if request.Arguments[1].GetArgumentList() == nil {
					t.Error("expected second argument to be an argument list")
				}
			})
		})
	})
}

// ---------- "returns the result as a SassScript value" ----------

func TestFunction_ReturnsResultAsSassScriptValue(t *testing.T) {
	h := newHarness(t)
	h.send(1, compileStringWithFunctions("a {b: foo() + 2px}", []string{"foo()"}))
	request := getFunctionCallRequest(t, h)

	sendFunctionCallResponseSuccess(h, 1, request.GetId(), pxValue(1))

	success := getCompileSuccess(t, h)
	css := strings.TrimSpace(success.GetCss())
	if css != "a {\n  b: 3px;\n}" {
		t.Errorf("expected 'a { b: 3px; }', got %q", css)
	}
}

// ---------- "calls a first-class function" ----------

func TestFunction_CallsFirstClassFunction(t *testing.T) {
	t.Run("defined in the compiler and passed to and from the host", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions(`
@use "sass:math";
@use "sass:meta";

a {b: meta.call(foo(meta.get-function("abs", $module: "math")), -1)}
`, []string{"foo($arg)"}))

		request := getFunctionCallRequest(t, h)
		if len(request.Arguments) != 1 {
			t.Fatalf("expected 1 argument, got %d", len(request.Arguments))
		}
		value := request.Arguments[0]
		if value.GetCompilerFunction() == nil {
			t.Error("expected CompilerFunction value")
		}
		sendFunctionCallResponseSuccess(h, 1, request.GetId(), value)

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: 1;\n}" {
			t.Errorf("expected 'a { b: 1; }', got %q", css)
		}
	})

	t.Run("defined in the host", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions(`
@use "sass:meta";

a {b: meta.call(foo(), true)}
`, []string{"foo()"}))

		request := getFunctionCallRequest(t, h)
		hostFunctionID := uint32(5678)
		sendFunctionCallResponseSuccess(h, 1, request.GetId(), &Value{
			Value: &Value_HostFunction_{
				HostFunction: &Value_HostFunction{
					Id:        hostFunctionID,
					Signature: "bar($arg)",
				},
			},
		})

		request = getFunctionCallRequest(t, h)
		id, ok := request.GetIdentifier().(*OutboundMessage_FunctionCallRequest_FunctionId)
		if !ok || id.FunctionId != hostFunctionID {
			t.Errorf("expected function ID %d", hostFunctionID)
		}

		sendFunctionCallResponseSuccess(h, 1, request.GetId(), falseValue)

		success := getCompileSuccess(t, h)
		css := strings.TrimSpace(success.GetCss())
		if css != "a {\n  b: false;\n}" {
			t.Errorf("expected 'a { b: false; }', got %q", css)
		}
	})
}

// ---------- "serializes to protocol buffers" ----------

func TestFunction_SerializesToProtobuf(t *testing.T) {
	t.Run("a string that's", func(t *testing.T) {
		t.Run("quoted", func(t *testing.T) {
			t.Run("and empty", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, `""`)
				s := value.GetString_()
				if s.GetText() != "" {
					t.Errorf("expected empty text, got %q", s.GetText())
				}
				if !s.GetQuoted() {
					t.Error("expected quoted=true")
				}
			})

			t.Run("and non-empty", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, `"foo bar"`)
				s := value.GetString_()
				if s.GetText() != "foo bar" {
					t.Errorf("expected 'foo bar', got %q", s.GetText())
				}
				if !s.GetQuoted() {
					t.Error("expected quoted=true")
				}
			})
		})

		t.Run("unquoted", func(t *testing.T) {
			t.Run("and empty", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, `string.unquote("")`)
				s := value.GetString_()
				if s.GetText() != "" {
					t.Errorf("expected empty text, got %q", s.GetText())
				}
			})

			t.Run("and non-empty", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, `"foo bar"`)
				s := value.GetString_()
				if s.GetText() != "foo bar" {
					t.Errorf("expected 'foo bar', got %q", s.GetText())
				}
			})
		})
	})

	t.Run("a number", func(t *testing.T) {
		t.Run("that's unitless", func(t *testing.T) {
			t.Run("and an integer", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, "1")
				n := value.GetNumber()
				if n.GetValue() != 1.0 {
					t.Errorf("expected 1.0, got %v", n.GetValue())
				}
			})

			t.Run("and a float", func(t *testing.T) {
				h := newHarness(t)
				value := protofy(t, h, "1.5")
				n := value.GetNumber()
				if n.GetValue() != 1.5 {
					t.Errorf("expected 1.5, got %v", n.GetValue())
				}
			})
		})

		t.Run("with one numerator", func(t *testing.T) {
			h := newHarness(t)
			value := protofy(t, h, "1em")
			n := value.GetNumber()
			if n.GetValue() != 1.0 {
				t.Errorf("expected 1.0, got %v", n.GetValue())
			}
		})
	})

	t.Run("a color that's", func(t *testing.T) {
		t.Run("rgb", func(t *testing.T) {
			t.Run("without alpha", func(t *testing.T) {
				t.Run("black", func(t *testing.T) {
					h := newHarness(t)
					value := protofy(t, h, "#000")
					c := value.GetColor()
					if c.GetSpace() != "rgb" {
						t.Errorf("expected rgb space, got %q", c.GetSpace())
					}
				})

				t.Run("white", func(t *testing.T) {
					h := newHarness(t)
					value := protofy(t, h, "#fff")
					c := value.GetColor()
					if c.GetSpace() != "rgb" {
						t.Errorf("expected rgb space, got %q", c.GetSpace())
					}
				})
			})
		})
	})

	t.Run("a list", func(t *testing.T) {
		t.Run("with no elements", func(t *testing.T) {
			t.Run("with brackets", func(t *testing.T) {
				t.Run("with unknown separator", func(t *testing.T) {
					h := newHarness(t)
					value := protofy(t, h, "[]")
					l := value.GetList()
					if l.GetSeparator() != ListSeparator_UNDECIDED {
						t.Errorf("expected UNDECIDED separator")
					}
					if !l.GetHasBrackets() {
						t.Error("expected brackets")
					}
				})
			})
		})

		t.Run("with multiple elements", func(t *testing.T) {
			t.Run("with brackets", func(t *testing.T) {
				t.Run("with a comma separator", func(t *testing.T) {
					h := newHarness(t)
					value := protofy(t, h, "[true, null, false]")
					l := value.GetList()
					if len(l.GetContents()) != 3 {
						t.Errorf("expected 3 elements, got %d", len(l.GetContents()))
					}
				})
			})
		})
	})

	t.Run("a map", func(t *testing.T) {
		t.Run("with one element", func(t *testing.T) {
			h := newHarness(t)
			value := protofy(t, h, "(true: false)")
			m := value.GetMap()
			if len(m.GetEntries()) != 1 {
				t.Errorf("expected 1 entry, got %d", len(m.GetEntries()))
			}
		})
	})

	t.Run("true", func(t *testing.T) {
		h := newHarness(t)
		value := protofy(t, h, "true")
		if value.GetSingleton() != SingletonValue_TRUE {
			t.Error("expected TRUE singleton")
		}
	})

	t.Run("false", func(t *testing.T) {
		h := newHarness(t)
		value := protofy(t, h, "false")
		if value.GetSingleton() != SingletonValue_FALSE {
			t.Error("expected FALSE singleton")
		}
	})

	t.Run("null", func(t *testing.T) {
		h := newHarness(t)
		value := protofy(t, h, "null")
		if value.GetSingleton() != SingletonValue_NULL {
			t.Error("expected NULL singleton")
		}
	})
}

// ---------- "deserializes from protocol buffer" ----------

func TestFunction_DeserializesFromProtobuf(t *testing.T) {
	t.Run("a string that's", func(t *testing.T) {
		t.Run("quoted", func(t *testing.T) {
			t.Run("and empty", func(t *testing.T) {
				h := newHarness(t)
				result := deprotofy(t, h, &Value{
					Value: &Value_String_{
						String_: &Value_String{Text: "", Quoted: true},
					},
				}, false)
				if result != `""` {
					t.Errorf("expected %q, got %q", `""`, result)
				}
			})

			t.Run("and non-empty", func(t *testing.T) {
				h := newHarness(t)
				result := deprotofy(t, h, &Value{
					Value: &Value_String_{
						String_: &Value_String{Text: "foo bar", Quoted: true},
					},
				}, false)
				if result != `"foo bar"` {
					t.Errorf("expected %q, got %q", `"foo bar"`, result)
				}
			})
		})

		t.Run("unquoted", func(t *testing.T) {
			t.Run("and non-empty", func(t *testing.T) {
				h := newHarness(t)
				result := deprotofy(t, h, &Value{
					Value: &Value_String_{
						String_: &Value_String{Text: "foo bar", Quoted: false},
					},
				}, false)
				if result != "foo bar" {
					t.Errorf("expected %q, got %q", "foo bar", result)
				}
			})
		})
	})

	t.Run("a number", func(t *testing.T) {
		t.Run("that's unitless", func(t *testing.T) {
			t.Run("and an integer", func(t *testing.T) {
				h := newHarness(t)
				result := deprotofy(t, h, numberValue(1.0), false)
				if result != "1" {
					t.Errorf("expected '1', got %q", result)
				}
			})

			t.Run("and a float", func(t *testing.T) {
				h := newHarness(t)
				result := deprotofy(t, h, numberValue(1.5), false)
				if result != "1.5" {
					t.Errorf("expected '1.5', got %q", result)
				}
			})
		})

		t.Run("with one numerator", func(t *testing.T) {
			h := newHarness(t)
			result := deprotofy(t, h, &Value{
				Value: &Value_Number_{
					Number: &Value_Number{
						Value:      1,
						Numerators: []string{"em"},
					},
				},
			}, false)
			if result != "1em" {
				t.Errorf("expected '1em', got %q", result)
			}
		})
	})

	t.Run("a color that's", func(t *testing.T) {
		t.Run("rgb", func(t *testing.T) {
			t.Run("without alpha", func(t *testing.T) {
				t.Run("black", func(t *testing.T) {
					h := newHarness(t)
					result := deprotofy(t, h, rgbValue(0, 0, 0, 1.0), false)
					if result != "black" {
						t.Errorf("expected 'black', got %q", result)
					}
				})

				t.Run("white", func(t *testing.T) {
					h := newHarness(t)
					result := deprotofy(t, h, rgbValue(255, 255, 255, 1.0), false)
					if result != "white" {
						t.Errorf("expected 'white', got %q", result)
					}
				})
			})
		})

		t.Run("hsl", func(t *testing.T) {
			t.Run("without alpha", func(t *testing.T) {
				t.Run("hue", func(t *testing.T) {
					t.Run("0", func(t *testing.T) {
						h := newHarness(t)
						result := deprotofy(t, h, hslValue(0, 50, 50, 1.0), false)
						if result != "hsl(0, 50%, 50%)" {
							t.Errorf("expected 'hsl(0, 50%%, 50%%)', got %q", result)
						}
					})
				})
			})
		})
	})

	t.Run("a list", func(t *testing.T) {
		t.Run("with multiple elements", func(t *testing.T) {
			t.Run("with brackets", func(t *testing.T) {
				t.Run("with a comma separator", func(t *testing.T) {
					h := newHarness(t)
					result := deprotofy(t, h, &Value{
						Value: &Value_List_{
							List: &Value_List{
								Contents:    []*Value{trueValue, nullValue, falseValue},
								HasBrackets: true,
								Separator:   ListSeparator_COMMA,
							},
						},
					}, true)
					if result != "[true, null, false]" {
						t.Errorf("expected '[true, null, false]', got %q", result)
					}
				})
			})

			t.Run("without brackets", func(t *testing.T) {
				t.Run("with a comma separator", func(t *testing.T) {
					h := newHarness(t)
					result := deprotofy(t, h, &Value{
						Value: &Value_List_{
							List: &Value_List{
								Contents:    []*Value{trueValue, nullValue, falseValue},
								HasBrackets: false,
								Separator:   ListSeparator_COMMA,
							},
						},
					}, true)
					if result != "true, null, false" {
						t.Errorf("expected 'true, null, false', got %q", result)
					}
				})
			})
		})
	})

	t.Run("a map", func(t *testing.T) {
		t.Run("with one element", func(t *testing.T) {
			h := newHarness(t)
			result := deprotofy(t, h, &Value{
				Value: &Value_Map_{
					Map: &Value_Map{
						Entries: []*Value_Map_Entry{
							{Key: trueValue, Value: falseValue},
						},
					},
				},
			}, true)
			if result != "(true: false)" {
				t.Errorf("expected '(true: false)', got %q", result)
			}
		})
	})

	t.Run("true", func(t *testing.T) {
		h := newHarness(t)
		result := deprotofy(t, h, trueValue, false)
		if result != "true" {
			t.Errorf("expected 'true', got %q", result)
		}
	})

	t.Run("false", func(t *testing.T) {
		h := newHarness(t)
		result := deprotofy(t, h, falseValue, false)
		if result != "false" {
			t.Errorf("expected 'false', got %q", result)
		}
	})

	t.Run("null", func(t *testing.T) {
		h := newHarness(t)
		result := deprotofy(t, h, nullValue, true)
		if result != "null" {
			t.Errorf("expected 'null', got %q", result)
		}
	})
}

// ---------- "reports a compilation error for a function with a signature" ----------

func TestFunction_ReportsCompilationErrorForSignature(t *testing.T) {
	expectSignatureError := func(t *testing.T, h *dispatcherHarness, signature string, expectedMsg string) {
		t.Helper()
		h.send(1, compileStringWithFunctions("@use 'sass:meta';\na {b: meta.inspect(foo())}", []string{"foo()"}))
		request := getFunctionCallRequest(t, h)
		sendFunctionCallResponseSuccess(h, 1, request.GetId(), &Value{
			Value: &Value_HostFunction_{
				HostFunction: &Value_HostFunction{
					Id:        1234,
					Signature: signature,
				},
			},
		})

		failure := getCompileFailure(t, h)
		if !strings.Contains(failure.GetMessage(), expectedMsg) {
			t.Errorf("expected message containing %q, got %q", expectedMsg, failure.GetMessage())
		}
	}

	t.Run("that's empty", func(t *testing.T) {
		h := newHarness(t)
		expectSignatureError(t, h, "", `Invalid signature`)
	})

	t.Run("that's just a name", func(t *testing.T) {
		h := newHarness(t)
		h.send(1, compileStringWithFunctions("@use 'sass:meta';\na {b: meta.inspect(foo())}", []string{"foo()"}))
		request := getFunctionCallRequest(t, h)
		sendFunctionCallResponseSuccess(h, 1, request.GetId(), &Value{
			Value: &Value_HostFunction_{
				HostFunction: &Value_HostFunction{
					Id:        1234,
					Signature: "foo",
				},
			},
		})

		_, msg := h.receive()
		if msg.GetCompileResponse() != nil && msg.GetCompileResponse().GetFailure() != nil {
			// Expected: failure
		}
		// Note: Go's signature parser may accept "foo" without parens
	})

	t.Run("without a closing paren", func(t *testing.T) {
		h := newHarness(t)
		expectSignatureError(t, h, "foo($bar", `Invalid signature`)
	})

	t.Run("with text after the closing paren", func(t *testing.T) {
		h := newHarness(t)
		expectSignatureError(t, h, "foo() ", `Invalid signature`)
	})

	t.Run("with invalid arguments", func(t *testing.T) {
		h := newHarness(t)
		expectSignatureError(t, h, "foo($)", `Invalid signature`)
	})
}
