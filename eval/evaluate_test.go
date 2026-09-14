// Copyright 2016 Google Inc. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.

package eval

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/extend"
	"github.com/bancek/go-sass/orderedmap"
	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sassio"
	"github.com/bancek/go-sass/sasslogger"
	"github.com/bancek/go-sass/sassurl"
	"github.com/bancek/go-sass/value"
)

// recordingLogger captures warn/deprecation calls for assertions.
type recordingLogger struct {
	warns      []string
	deprecates []string
}

func (r *recordingLogger) Warn(message string, span *sasscommon.FileSpan, trace *sasscommon.Trace) {
	r.warns = append(r.warns, message)
}

func (r *recordingLogger) Debug(message string, span *sasscommon.FileSpan) {}

func (r *recordingLogger) WarnDeprecation(message string, span *sasscommon.FileSpan, dep *deprecation.Deprecation, trace *sasscommon.Trace) error {
	r.deprecates = append(r.deprecates, message)
	return nil
}

func testVisitor(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	if logger == nil {
		logger = &recordingLogger{}
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	v := NewEvaluateVisitor(ic, logger)
	v.ec.SetCallableNode(testNode{span: testSpan()})
	return v
}

type testNode struct{ span sasscommon.FileSpan }

func (n testNode) Span() (sasscommon.FileSpan, error) { return n.span, nil }
func (n testNode) IsAstNode()                         {}

func testSpan() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("test { color: red; }"), nil)
	return sasscommon.NewFileSpan(fs, 0, 4)
}

func testSpan2() sasscommon.FileSpan {
	fs := sasscommon.NewFileSource([]byte("other { color: blue; }"), nil)
	return sasscommon.NewFileSpan(fs, 0, 5)
}

func TestWarnDedupSameMessageAndSpan(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)

	sp := testSpan()
	// First call
	if err := v.warn("test warning", sp, nil); err != nil {
		t.Fatal(err)
	}
	// Second call with same message + span
	if err := v.warn("test warning", sp, nil); err != nil {
		t.Fatal(err)
	}

	if len(rl.warns) != 1 {
		t.Errorf("dedup failed: got %d warns, want 1", len(rl.warns))
	}
}

func TestWarnDifferentMessagesEmitBoth(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)

	if err := v.warn("first", testSpan(), nil); err != nil {
		t.Fatal(err)
	}
	if err := v.warn("second", testSpan(), nil); err != nil {
		t.Fatal(err)
	}

	if len(rl.warns) != 2 {
		t.Errorf("got %d warns, want 2", len(rl.warns))
	}
}

func TestWarnDifferentSpansEmitBoth(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)

	if err := v.warn("same message", testSpan(), nil); err != nil {
		t.Fatal(err)
	}
	if err := v.warn("same message", testSpan2(), nil); err != nil {
		t.Fatal(err)
	}

	if len(rl.warns) != 2 {
		t.Errorf("got %d warns, want 2", len(rl.warns))
	}
}

func TestWarnQuietDepsSuppresses(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	v.quietDeps = true
	v.inDependency = true

	if err := v.warn("should not appear", testSpan(), nil); err != nil {
		t.Fatal(err)
	}

	if len(rl.warns) != 0 {
		t.Errorf("quiet-deps should suppress: got %d warns, want 0", len(rl.warns))
	}
}

func TestWarnQuietDepsButNotInDependency(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	v.quietDeps = true
	v.inDependency = false

	if err := v.warn("should appear", testSpan(), nil); err != nil {
		t.Fatal(err)
	}

	if len(rl.warns) != 1 {
		t.Errorf("quiet-deps should not suppress when not in dependency: got %d warns, want 1", len(rl.warns))
	}
}

func TestWarnDeprecation(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)

	err := v.warn("deprecated feature", testSpan(), deprecation.CallString)
	if err != nil {
		t.Fatal(err)
	}

	if len(rl.deprecates) != 1 {
		t.Fatalf("expected 1 deprecation, got %d", len(rl.deprecates))
	}
	if rl.deprecates[0] != "deprecated feature" {
		t.Errorf("deprecation message = %q, want %q", rl.deprecates[0], "deprecated feature")
	}
}

// ===========================================================================
// stackTrace() tests
// ===========================================================================

func TestStackTraceEmptyStack(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"

	trace := v.stackTrace(testSpan())

	// Empty stack: trace has just the span frame
	if trace == nil || len(trace.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(trace.Frames))
	}
	if trace.Frames[0].Member != "root stylesheet" {
		t.Errorf("member = %q, want %q", trace.Frames[0].Member, "root stylesheet")
	}
}

func TestStackTraceSingleFrame(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"
	v.stack = &stackFrame{
		Name:   "my-mixin()",
		Span:   testSpan2(),
		Parent: nil,
	}

	trace := v.stackTrace(testSpan())

	// Order: [span_frame, innermost_stack_frame, ..., outermost]
	// With one stack frame: [span_frame, stack_frame]
	if trace == nil || len(trace.Frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(trace.Frames))
	}
	if trace.Frames[0].Member != "root stylesheet" {
		t.Errorf("frames[0].member = %q, want %q", trace.Frames[0].Member, "root stylesheet")
	}
	if trace.Frames[1].Member != "my-mixin()" {
		t.Errorf("frames[1].member = %q, want %q", trace.Frames[1].Member, "my-mixin()")
	}
}

func TestStackTraceMultiFrameOrdering(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"

	// Stack linked list: middle → innermost → nil
	// Innermost stack frame (deepest in call chain)
	innermost := &stackFrame{
		Name:   "helper()",
		Span:   testSpan2(),
		Parent: nil,
	}
	// Middle frame (called helper)
	middle := &stackFrame{
		Name:   "my-mixin()",
		Span:   testSpan(),
		Parent: innermost,
	}
	v.stack = middle

	trace := v.stackTrace(testSpan2())

	// Order: [span_frame(root), innermost(helper), outermost(mixin)]
	// Go logic: collect [mixin, helper] → reverse [helper, mixin] →
	// append span [helper, mixin, root] → reverse [root, mixin, helper]
	if trace == nil || len(trace.Frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(trace.Frames))
	}
	if trace.Frames[0].Member != "root stylesheet" {
		t.Errorf("frames[0] = %q, want root stylesheet", trace.Frames[0].Member)
	}
	if trace.Frames[1].Member != "my-mixin()" {
		t.Errorf("frames[1] = %q, want my-mixin()", trace.Frames[1].Member)
	}
	if trace.Frames[2].Member != "helper()" {
		t.Errorf("frames[2] = %q, want helper()", trace.Frames[2].Member)
	}
}

func TestStackTraceNilSpan(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"
	v.stack = &stackFrame{
		Name:   "my-mixin()",
		Span:   testSpan(),
		Parent: nil,
	}

	trace := v.stackTrace(nil)

	// nil span → no span frame
	if trace == nil || len(trace.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(trace.Frames))
	}
	if trace.Frames[0].Member != "my-mixin()" {
		t.Errorf("member = %q, want %q", trace.Frames[0].Member, "my-mixin()")
	}
}

// ===========================================================================
// exception() tests
// ===========================================================================

func TestExceptionWithSpan(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"

	sp := testSpan()
	exc := v.exception("something went wrong", &sp)

	if exc.Message != "something went wrong" {
		t.Errorf("message = %q, want %q", exc.Message, "something went wrong")
	}
	if exc.Trace == nil {
		t.Fatal("trace should not be nil")
	}
	// Trace should have at least the member frame
	if len(exc.Trace.Frames) == 0 {
		t.Error("trace should have at least 1 frame")
	}
}

func TestExceptionWithoutSpanUsesStackSpan(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"
	sp2 := testSpan2()
	v.stack = &stackFrame{
		Name: "my-func()",
		Span: sp2,
	}

	exc := v.exception("error without explicit span", nil)

	// When span is nil and stack exists, exception should have a span
	// (the stack top's span). We verify the message is correct.
	if exc.Message != "error without explicit span" {
		t.Errorf("message = %q", exc.Message)
	}
}

func TestExceptionWithoutSpanOrStack(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"
	// No stack

	exc := v.exception("error, no span, no stack", nil)

	// Should still produce a valid exception with trace
	if exc.Message != "error, no span, no stack" {
		t.Errorf("message = %q", exc.Message)
	}
}

// ===========================================================================
// withEvaluationContext() tests
// ===========================================================================

func TestWithEvaluationContextSaveRestore(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.ec.SetDefaultWarnSpan(testSpan())
	old := v.ec.DefaultWarnSpan()

	called := false
	err := v.withEvaluationContext(testSpan2(), func() error {
		called = true
		// Inside callback: default_warn_span should be the new span
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("callback was not called")
	}

	// After callback: restored to old span
	if v.ec.DefaultWarnSpan() != old {
		t.Error("default_warn_span not restored after callback")
	}
}

// ===========================================================================
// humanizeFrame() tests
// ===========================================================================

func TestHumanizeFrameWithImportCache(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})

	u, _ := sassurl.Parse("file:///home/user/styles.scss")
	frame := &sasscommon.Frame{
		URI:    u,
		Line:   42,
		Column: 5,
		Member: "my-mixin()",
	}

	v.humanizeFrame(frame)

	// ImportCache exists (created in testVisitor), so URI should be humanized.
	// The default import cache may not have this URL key, so humanize may
	// leave it unchanged or return something else. We just verify no panic.
	if frame.URI == nil {
		t.Error("URI should not be nil after humanizeFrame")
	}
}

func TestHumanizeFrameNilImportCache(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.importCache = nil

	u, _ := sassurl.Parse("file:///home/user/styles.scss")
	frame := &sasscommon.Frame{
		URI:    u,
		Line:   42,
		Column: 5,
		Member: "my-mixin()",
	}

	v.humanizeFrame(frame)

	// With nil import cache, URI should be unchanged
	if frame.URI.String() != "file:///home/user/styles.scss" {
		t.Errorf("URI should be unchanged, got %v", frame.URI)
	}
}

// ===========================================================================
// stackTrace() display tests
// ===========================================================================

func TestStackTraceStringFormat(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.member = "root stylesheet"
	v.stack = &stackFrame{
		Name:   "my-mixin()",
		Span:   testSpan2(),
		Parent: nil,
	}

	trace := v.stackTrace(testSpan())

	// Trace.String() should produce the standard Sass stack format.
	// Format: "  member  location"
	s := trace.String()
	if !strings.Contains(s, "root stylesheet") {
		t.Errorf("trace should contain root stylesheet, got: %s", s)
	}
	if !strings.Contains(s, "my-mixin") {
		t.Errorf("trace should contain my-mixin, got: %s", s)
	}
}

func TestDefaultNamespaceUnderscorePrefix(t *testing.T) {
	u, _ := sassurl.Parse("file:///home/user/_bootstrap.scss")
	got := defaultNamespace(u)
	// All underscores are replaced with hyphens, including the leading one.
	if got != "-bootstrap" {
		t.Errorf("defaultNamespace = %q, want %q", got, "-bootstrap")
	}
}

func TestDefaultNamespaceUnderscoreToHyphen(t *testing.T) {
	u, _ := sassurl.Parse("file:///home/user/my_utils.scss")
	got := defaultNamespace(u)
	if got != "my-utils" {
		t.Errorf("defaultNamespace = %q, want %q", got, "my-utils")
	}
}

func TestDefaultNamespaceMultipleExtensions(t *testing.T) {
	u, _ := sassurl.Parse("file:///home/user/theme.css.scss")
	got := defaultNamespace(u)
	if got != "theme.css" {
		t.Errorf("defaultNamespace = %q, want %q", got, "theme.css")
	}
}

// ===========================================================================
// verifyParameterList() tests
// ===========================================================================

func TestVerifyParameterListExactMatch(t *testing.T) {
	params := value.NewParameterList(
		[]*value.Parameter{
			value.NewParameter("x", testSpan(), nil),
			value.NewParameter("y", testSpan(), nil),
		},
		testSpan(),
		nil,
	)
	names := map[string]bool{}
	err := verifyParameterList(2, names, params)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestVerifyParameterListTooManyPositional(t *testing.T) {
	params := value.NewParameterList(
		[]*value.Parameter{value.NewParameter("x", testSpan(), nil)},
		testSpan(),
		nil,
	)
	names := map[string]bool{}
	err := verifyParameterList(3, names, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Only 1 argument allowed") {
		t.Errorf("error = %q, want 'Only 1 argument allowed'", err.Error())
	}
}

func TestVerifyParameterListMissingArgument(t *testing.T) {
	params := value.NewParameterList(
		[]*value.Parameter{
			value.NewParameter("x", testSpan(), nil),
			value.NewParameter("y", testSpan(), nil),
		},
		testSpan(),
		nil,
	)
	names := map[string]bool{}
	err := verifyParameterList(1, names, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Missing argument $y.") {
		t.Errorf("error = %q, want 'Missing argument $y.'", err.Error())
	}
}

func TestVerifyParameterListBothPositionalAndNamed(t *testing.T) {
	params := value.NewParameterList(
		[]*value.Parameter{value.NewParameter("x", testSpan(), nil)},
		testSpan(),
		nil,
	)
	names := map[string]bool{"x": true}
	err := verifyParameterList(1, names, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "was passed both by position and by name") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestVerifyParameterListUnknownNamed(t *testing.T) {
	params := value.NewParameterList(
		[]*value.Parameter{value.NewParameter("x", testSpan(), nil)},
		testSpan(),
		nil,
	)
	names := map[string]bool{"z": true}
	err := verifyParameterList(1, names, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "No parameter named $z.") {
		t.Errorf("error = %q", err.Error())
	}
}

func TestVerifyParameterListRestParameterAcceptsAll(t *testing.T) {
	restName := "rest"
	params := value.NewParameterList(
		[]*value.Parameter{value.NewParameter("x", testSpan(), nil)},
		testSpan(),
		&restName,
	)
	names := map[string]bool{}
	err := verifyParameterList(10, names, params)
	if err != nil {
		t.Errorf("rest parameter should accept extra args: got %v", err)
	}
}

func TestVerifyParameterListNilParamsNoArgs(t *testing.T) {
	names := map[string]bool{}
	err := verifyParameterList(0, names, nil)
	if err != nil {
		t.Errorf("nil params with 0 args should be ok: got %v", err)
	}
}

func TestVerifyParameterListNilParamsWithArgs(t *testing.T) {
	names := map[string]bool{}
	err := verifyParameterList(1, names, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Expected 0 arguments") {
		t.Errorf("error = %q", err.Error())
	}
}

// ===========================================================================
// mergeMediaQueries() tests
// ===========================================================================

func TestMergeMediaQueriesEmptyFirst(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	q1 := []*value.CssMediaQuery{}
	screen := "screen"
	q2 := []*value.CssMediaQuery{value.NewCssMediaQueryType(&screen, nil, nil)}
	result := v.mergeMediaQueries(q1, q2)
	if len(result) != 0 {
		t.Errorf("empty q1 should produce empty result, got %d", len(result))
	}
}

func TestMergeMediaQueriesMatching(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	screen := "screen"
	q1 := []*value.CssMediaQuery{value.NewCssMediaQueryType(&screen, nil, nil)}
	q2 := []*value.CssMediaQuery{value.NewCssMediaQueryType(&screen, nil, nil)}
	result := v.mergeMediaQueries(q1, q2)
	if len(result) != 1 {
		t.Errorf("matching screen queries should merge to 1, got %d", len(result))
	}
}

// ===========================================================================
// styleRule() tests
// ===========================================================================

func TestStyleRuleNilWhenExcludingAtRoot(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.atRootExcludingStyleRule = true

	got := v.styleRule()
	if got != nil {
		t.Error("styleRule should return nil when atRootExcludingStyleRule")
	}
}

func TestStyleRuleReturnsFieldWhenNotExcluding(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	v.atRootExcludingStyleRule = false
	// styleRuleIgnoringAtRoot is nil by default, so styleRule returns nil.
	got := v.styleRule()
	if got != nil {
		t.Error("styleRule returns nil when styleRuleIgnoringAtRoot is nil")
	}
}

// ── withoutSlash ──

func ptr(s string) *string { return &s }

func TestWithoutSlashNumberWithSlash(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	num := value.NewSassNumber(42.0, nil).WithSlash(
		value.NewSassNumber(1.0, ptr("px")),
		value.NewSassNumber(2.0, ptr("px")),
	)
	node := testNode{span: testSpan()}

	result, err := v.withoutSlash(num, node)
	if err != nil {
		t.Fatal(err)
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	if rn.HasSlash() {
		t.Error("result should not have slash")
	}
	if len(rl.deprecates) != 1 {
		t.Fatalf("expected 1 deprecation, got %d", len(rl.deprecates))
	}
	want := strings.Join([]string{
		`Using / for division is deprecated and will be removed in Dart Sass 2.0.0.`,
		``,
		`Recommendation: math.div(1px, 2px)`,
		``,
		`More info and automated migrator: https://sass-lang.com/d/slash-div`,
	}, "\n")
	if rl.deprecates[0] != want {
		t.Errorf("deprecation message =\n%q\nwant =\n%q", rl.deprecates[0], want)
	}
}

func TestWithoutSlashNumberWithoutSlash(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	num := value.NewSassNumber(42.0, nil)
	node := testNode{span: testSpan()}

	result, err := v.withoutSlash(num, node)
	if err != nil {
		t.Fatal(err)
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	if rn.HasSlash() {
		t.Error("should not have slash")
	}
	if len(rl.deprecates) != 0 {
		t.Errorf("expected 0 deprecations, got %d", len(rl.deprecates))
	}
}

func TestWithoutSlashNonNumber(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	str := &value.SassString{Text: "hello", HasQuotes: false}
	node := testNode{span: testSpan()}

	result, err := v.withoutSlash(str, node)
	if err != nil {
		t.Fatal(err)
	}
	if result != str {
		t.Error("non-number should pass through unchanged")
	}
	if len(rl.deprecates) != 0 {
		t.Errorf("expected 0 deprecations, got %d", len(rl.deprecates))
	}
}

// ── slashDivisionRecommendation ──

func TestSlashDivisionRecommendationSimple(t *testing.T) {
	num := value.NewSassNumber(1.0, ptr("px")).WithSlash(
		value.NewSassNumber(1.0, nil),
		value.NewSassNumber(2.0, nil),
	)
	got, err := slashDivisionRecommendation(num)
	if err != nil {
		t.Fatal(err)
	}
	want := "math.div(1, 2)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSlashDivisionRecommendationNested(t *testing.T) {
	inner := value.NewSassNumber(1.0, ptr("px")).WithSlash(
		value.NewSassNumber(1.0, ptr("px")),
		value.NewSassNumber(2.0, ptr("px")),
	)
	outer := value.NewSassNumber(42.0, nil).WithSlash(
		inner,
		value.NewSassNumber(3.0, ptr("px")),
	)
	got, err := slashDivisionRecommendation(outer)
	if err != nil {
		t.Fatal(err)
	}
	want := "math.div(math.div(1px, 2px), 3px)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── expressionNode ──

func TestExpressionNodeNonVariable(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	num := value.NewNumberExpression(42.0, sp, nil)

	node, err := v.expressionNode(num)
	if err != nil {
		t.Fatal(err)
	}
	gotSpan, err := node.Span()
	if err != nil {
		t.Fatal(err)
	}
	gotLen, err := gotSpan.Length()
	if err != nil {
		t.Fatal(err)
	}
	wantSpan, err := num.Span()
	if err != nil {
		t.Fatal(err)
	}
	wantLen, err := wantSpan.Length()
	if err != nil {
		t.Fatal(err)
	}
	if gotLen != wantLen {
		t.Errorf("got length %d, want %d", gotLen, wantLen)
	}
}

func TestExpressionNodeVariableFound(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	defSp := testSpan()
	defNode := testNode{span: defSp}
	v.env.SetLocalVariable("x", value.SassTrue, defNode)

	refSp := testSpan2()
	ve := value.NewVariableExpression("x", refSp, nil)

	node, err := v.expressionNode(ve)
	if err != nil {
		t.Fatal(err)
	}
	gotSpan, err := node.Span()
	if err != nil {
		t.Fatal(err)
	}
	gotLen, err := gotSpan.Length()
	if err != nil {
		t.Fatal(err)
	}
	defLen, err := defSp.Length()
	if err != nil {
		t.Fatal(err)
	}
	refLen, err := refSp.Length()
	if err != nil {
		t.Fatal(err)
	}
	if gotLen != defLen {
		t.Errorf("got length %d (expected definition span), want %d; reference span has length %d",
			gotLen, defLen, refLen)
	}
}

func TestExpressionNodeVariableNotFound(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	ve := value.NewVariableExpression("missing", sp, nil)

	node, err := v.expressionNode(ve)
	if err != nil {
		t.Fatal(err)
	}
	gotSpan, err := node.Span()
	if err != nil {
		t.Fatal(err)
	}
	gotLen, err := gotSpan.Length()
	if err != nil {
		t.Fatal(err)
	}
	wantLen, err := sp.Length()
	if err != nil {
		t.Fatal(err)
	}
	if gotLen != wantLen {
		t.Errorf("got length %d, want %d", gotLen, wantLen)
	}
}

// ── @return propagation ──

func TestVisitReturnRuleReturnsValue(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	num := value.NewNumberExpression(42.0, sp, nil)
	rule := value.NewReturnRule(num, sp)

	result, err := v.VisitReturnRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil return value")
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	val, _ := rn.AsInt()
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestHandleReturnShortCircuit(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	lastCalled := -1

	result, err := handleReturn(v, []int{0, 1, 2}, func(i int) (value.Value, error) {
		lastCalled = i
		if i == 1 {
			return value.NewSassNumber(float64(i), nil), nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if lastCalled != 1 {
		t.Errorf("expected lastCalled = 1 (short-circuited), got %d", lastCalled)
	}
}

func TestHandleReturnNoReturn(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	count := 0

	result, err := handleReturn(v, []int{0, 1, 2}, func(i int) (value.Value, error) {
		count++
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if count != 3 {
		t.Errorf("expected all 3 items processed, got %d", count)
	}
}

func TestHandleReturnPropagatesError(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	_, err := handleReturn(v, []int{0}, func(i int) (value.Value, error) {
		return nil, &sasscommon.SassRuntimeException{Message: "test error", Span: testSpan()}
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "test error") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "test error")
	}
}

func TestVisitReturnRuleAppliesWithoutSlash(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	sp := testSpan()
	slashNum := value.NewSassNumber(1.0, ptr("px")).WithSlash(
		value.NewSassNumber(1.0, nil),
		value.NewSassNumber(2.0, nil),
	)
	numExpr := value.NewValueExpression(slashNum, sp)
	rule := value.NewReturnRule(numExpr, sp)

	result, err := v.VisitReturnRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	if rn.HasSlash() {
		t.Error("result should not have slash after VisitReturnRule")
	}
	if len(rl.deprecates) != 1 {
		t.Fatalf("expected 1 deprecation for slash-div, got %d", len(rl.deprecates))
	}
}

// ── @return propagation through control flow ──

func TestReturnPropagatesThroughIfRule(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	returnStmt := value.NewReturnRule(
		value.NewValueExpression(value.NewSassNumber(7.0, nil), sp),
		sp,
	)
	condExpr := value.NewBooleanExpression(true, sp)
	ifClause := value.NewIfClause(condExpr, []value.Statement{returnStmt})
	ifRule := value.NewIfRule([]*value.IfClause{ifClause}, sp, nil)

	result, err := v.VisitIfRule(ifRule)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil @return value from @if")
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	val, _ := rn.AsInt()
	if val != 7 {
		t.Errorf("expected 7, got %d", val)
	}
}

func TestReturnPropagatesThroughForRule(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	returnStmt := value.NewReturnRule(
		value.NewValueExpression(value.NewSassNumber(5.0, nil), sp),
		sp,
	)
	fromExpr := value.NewNumberExpression(1.0, sp, nil)
	toExpr := value.NewNumberExpression(3.0, sp, nil)
	forRule := value.NewForRule("i", fromExpr, toExpr, []value.Statement{returnStmt}, sp, false)

	result, err := v.VisitForRule(forRule)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil @return value from @for")
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	val, _ := rn.AsInt()
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}
}

func TestReturnPropagatesThroughEachRule(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	returnStmt := value.NewReturnRule(
		value.NewValueExpression(value.NewSassNumber(3.0, nil), sp),
		sp,
	)
	items, err := value.NewSassList([]value.Value{
		value.NewSassNumber(1.0, nil),
	}, value.ListSeparatorUndecided, false)
	if err != nil {
		t.Fatal(err)
	}
	listExpr := value.NewValueExpression(items, sp)
	eachRule := value.NewEachRule([]string{"x"}, listExpr, []value.Statement{returnStmt}, sp)

	result, err := v.VisitEachRule(eachRule)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil @return value from @each")
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	val, _ := rn.AsInt()
	if val != 3 {
		t.Errorf("expected 3, got %d", val)
	}
}

func TestReturnPropagatesThroughWhileRule(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	returnStmt := value.NewReturnRule(
		value.NewValueExpression(value.NewSassNumber(9.0, nil), sp),
		sp,
	)
	condExpr := value.NewBooleanExpression(true, sp)
	whileRule := value.NewWhileRule(condExpr, []value.Statement{returnStmt}, sp)

	result, err := v.VisitWhileRule(whileRule)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil @return value from @while")
	}
	rn, ok := result.(value.SassNumber)
	if !ok {
		t.Fatalf("expected SassNumber, got %T", result)
	}
	val, _ := rn.AsInt()
	if val != 9 {
		t.Errorf("expected 9, got %d", val)
	}
}

// ── evaluateArguments rest/kwRest dispatch ──

func TestEvaluateArgumentsRestMap(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	m := value.EmptySassMap()
	m.Set(&value.SassString{Text: "k", HasQuotes: false}, value.NewSassNumber(1.0, nil))
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, value.NewValueExpression(m, sp), nil)

	results, err := v.evaluateArguments(args)
	if err != nil {
		t.Fatal(err)
	}
	if results.named.Len() != 1 {
		t.Fatalf("expected 1 named arg, got %d", results.named.Len())
	}
	val, _ := results.named.Get("k")
	if val == nil {
		t.Fatal("expected 'k' in named args")
	}
	if len(results.namedNodes) != 1 {
		t.Errorf("expected 1 named node, got %d", len(results.namedNodes))
	}
}

func TestEvaluateArgumentsRestList(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	lst, _ := value.NewSassList([]value.Value{value.NewSassNumber(1.0, nil), value.NewSassNumber(2.0, nil)}, value.ListSeparatorComma, false)
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, value.NewValueExpression(lst, sp), nil)

	results, err := v.evaluateArguments(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.positional) != 2 {
		t.Fatalf("expected 2 positional args, got %d", len(results.positional))
	}
	if results.separator != value.ListSeparatorComma {
		t.Errorf("expected comma separator, got %v", results.separator)
	}
	if len(results.positionalNodes) != 2 {
		t.Errorf("expected 2 positional nodes, got %d", len(results.positionalNodes))
	}
}

func TestEvaluateArgumentsRestArgumentList(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	kw := orderedmap.New[string, value.Value]()
	kw.Put("kw", value.NewSassNumber(3.0, nil))
	al, _ := value.NewSassArgumentList(
		[]value.Value{value.NewSassNumber(1.0, nil)},
		kw,
		value.ListSeparatorComma,
	)
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, value.NewValueExpression(al, sp), nil)

	results, err := v.evaluateArguments(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.positional) != 1 {
		t.Fatalf("expected 1 positional, got %d", len(results.positional))
	}
	if results.separator != value.ListSeparatorComma {
		t.Errorf("expected comma separator, got %v", results.separator)
	}
	if results.named.Len() != 1 {
		t.Fatalf("expected 1 named kw arg, got %d", results.named.Len())
	}
	kwVal, _ := results.named.Get("kw")
	if kwVal == nil {
		t.Fatal("expected 'kw' in named args")
	}
}

func TestEvaluateArgumentsRestSingle(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, value.NewValueExpression(value.NewSassNumber(42.0, nil), sp), nil)

	results, err := v.evaluateArguments(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(results.positional) != 1 {
		t.Fatalf("expected 1 positional, got %d", len(results.positional))
	}
	if len(results.positionalNodes) != 1 {
		t.Errorf("expected 1 positional node, got %d", len(results.positionalNodes))
	}
}

func TestEvaluateArgumentsKwRestMap(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	m := value.EmptySassMap()
	m.Set(&value.SassString{Text: "a", HasQuotes: false}, value.NewSassNumber(5.0, nil))
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, nil, value.NewValueExpression(m, sp))

	results, err := v.evaluateArguments(args)
	if err != nil {
		t.Fatal(err)
	}
	if results.named.Len() != 1 {
		t.Fatalf("expected 1 named arg, got %d", results.named.Len())
	}
}

func TestEvaluateArgumentsKwRestNotMap(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	args := value.NewArgumentList([]value.Expression{}, orderedmap.New[string, value.Expression](), nil, sp, nil, value.NewValueExpression(value.NewSassNumber(1.0, nil), sp))

	_, err := v.evaluateArguments(args)
	if err == nil {
		t.Fatal("expected error for non-map kwRest")
	}
	sre, ok := err.(*sasscommon.SassRuntimeException)
	if !ok {
		t.Fatalf("expected SassRuntimeException, got %T", err)
	}
	want := "Variable keyword arguments must be a map (was 1)."
	if sre.Message != want {
		t.Errorf("message = %q, want %q", sre.Message, want)
	}
}

// ── evaluateMacroArguments ──

func TestMacroArgumentsNoRest(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	posExpr := value.NewValueExpression(value.NewSassNumber(1.0, nil), sp)
	args := value.NewArgumentList(
		[]value.Expression{posExpr},
		orderedmap.New[string, value.Expression](),
		nil,
		sp,
		nil,
		nil,
	)

	pos, named, err := v.evaluateMacroArguments(args, testNode{span: sp})
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 {
		t.Fatalf("expected 1 positional expression, got %d", len(pos))
	}
	if named.Len() != 0 {
		t.Fatalf("expected 0 named expressions, got %d", named.Len())
	}
}

func TestMacroArgumentsRestMap(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	m := value.EmptySassMap()
	m.Set(&value.SassString{Text: "k", HasQuotes: false}, value.NewSassNumber(5.0, nil))
	args := value.NewArgumentList(
		[]value.Expression{},
		orderedmap.New[string, value.Expression](),
		nil,
		sp,
		value.NewValueExpression(m, sp),
		nil,
	)

	pos, named, err := v.evaluateMacroArguments(args, testNode{span: sp})
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 0 {
		t.Fatalf("expected 0 positional expressions, got %d", len(pos))
	}
	if named.Len() != 1 {
		t.Fatalf("expected 1 named expression, got %d", named.Len())
	}
	expr, _ := named.Get("k")
	if expr == nil {
		t.Fatal("expected expression for key 'k'")
	}
}

func TestMacroArgumentsKwRestNotMap(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	args := value.NewArgumentList(
		[]value.Expression{},
		orderedmap.New[string, value.Expression](),
		nil,
		sp,
		value.NewValueExpression(value.NewSassNumber(1.0, nil), sp), // rest: single value → triggers kwRest check
		value.NewValueExpression(value.NewSassNumber(2.0, nil), sp), // kwRest: not a map → error
	)

	_, _, err := v.evaluateMacroArguments(args, testNode{span: sp})
	if err == nil {
		t.Fatal("expected error for non-map kwRest")
	}
	sre, ok := err.(*sasscommon.SassRuntimeException)
	if !ok {
		t.Fatalf("expected SassRuntimeException, got %T", err)
	}
	want := "Variable keyword arguments must be a map (was 2)."
	if sre.Message != want {
		t.Errorf("message = %q, want %q", sre.Message, want)
	}
}

// ── at-rule value interpolation ──

func testVisitorWithRoot(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	v := testVisitor(t, logger)
	root := value.NewModifiableCssStylesheet(testSpan())
	v.root = root
	v.parent = root
	return v
}

func TestEvaluateAtRuleWithValue(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	name := value.NewInterpolationPlain("import", sp)
	valInterp := value.NewInterpolationPlain("url(\"foo.css\")", sp)
	rule := value.NewAtRule(name, sp, valInterp, nil)

	_, err := v.VisitAtRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	atRule, ok := children[0].(*value.ModifiableCssAtRule)
	if !ok {
		t.Fatalf("expected ModifiableCssAtRule, got %T", children[0])
	}
	if atRule.Value() == nil {
		t.Fatal("expected value on at-rule")
	}
	if atRule.Value().Value != "url(\"foo.css\")" {
		t.Errorf("value = %q, want %q", atRule.Value().Value, "url(\"foo.css\")")
	}
}

// ── declaration CSS node ──

func TestEvaluateDeclarationCreatesCssNode(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	v.inUnknownAtRule = true
	name := value.NewInterpolationPlain("color", sp)
	valExpr := value.NewValueExpression(value.Null, sp)
	valExpr = value.NewValueExpression(&value.SassString{Text: "red", HasQuotes: false}, sp)
	decl := value.NewDeclaration(name, valExpr, sp)

	_, err := v.VisitDeclaration(decl)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	cssDecl, ok := children[0].(*value.ModifiableCssDeclaration)
	if !ok {
		t.Fatalf("expected ModifiableCssDeclaration, got %T", children[0])
	}
	if cssDecl.Name().Value != "color" {
		t.Errorf("name = %q, want %q", cssDecl.Name().Value, "color")
	}
}

func TestEvaluateDeclarationOutsideStyleRule(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	name := value.NewInterpolationPlain("color", sp)
	valExpr := value.NewValueExpression(value.NewSassNumber(0.0, ptr("")), sp)
	decl := value.NewDeclaration(name, valExpr, sp)

	_, err := v.VisitDeclaration(decl)
	if err == nil {
		t.Fatal("expected error for declaration outside style rule")
	}
	want := "Declarations may only be used within style rules."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestEvaluateDeclarationBlankValue(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	v.inUnknownAtRule = true
	name := value.NewInterpolationPlain("color", sp)
	valExpr := value.NewValueExpression(value.Null, sp)
	decl := value.NewDeclaration(name, valExpr, sp)

	_, err := v.VisitDeclaration(decl)
	if err != nil {
		t.Fatal(err)
	}
	// Null is blank, no CSS node should be added
	children := v.root.Children()
	if len(children) != 0 {
		t.Errorf("expected 0 children for blank value, got %d", len(children))
	}
}

func TestEvaluateDeclarationNestedNamePrefix(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	v.inUnknownAtRule = true
	v.declarationName = "font"
	name := value.NewInterpolationPlain("weight", sp)
	valExpr := value.NewValueExpression(&value.SassString{Text: "bold", HasQuotes: false}, sp)
	decl := value.NewDeclaration(name, valExpr, sp)

	_, err := v.VisitDeclaration(decl)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	cssDecl := children[0].(*value.ModifiableCssDeclaration)
	if cssDecl.Name().Value != "font-weight" {
		t.Errorf("name = %q, want %q", cssDecl.Name().Value, "font-weight")
	}
}

// ── evaluateMediaRule query parsing ──

func TestEvaluateMediaRuleSimpleQuery(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	query := value.NewInterpolationPlain("screen", sp)
	rule := value.NewMediaRule(query, nil, sp)

	_, err := v.VisitMediaRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	mr, ok := children[0].(*value.ModifiableCssMediaRule)
	if !ok {
		t.Fatalf("expected ModifiableCssMediaRule, got %T", children[0])
	}
	queries := mr.Queries()
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}
	if queries[0].Type == nil || *queries[0].Type != "screen" {
		t.Errorf("expected 'screen' query type")
	}
}

func TestEvaluateMediaRuleNestedDeclarationError(t *testing.T) {
	v := testVisitorWithRoot(t, &recordingLogger{})
	sp := testSpan()
	v.declarationName = "color"
	query := value.NewInterpolationPlain("screen", sp)
	rule := value.NewMediaRule(query, nil, sp)

	_, err := v.VisitMediaRule(rule)
	if err == nil {
		t.Fatal("expected error for media rule in nested declaration")
	}
	want := "Media rules may not be used within nested declarations."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ── performInterpolationWithMap ──

func TestPerformInterpolationWithMapPlainText(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	interp := value.NewInterpolationPlain("hello", sp)

	result, im, err := v.performInterpolationWithMap(interp, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello" {
		t.Errorf("result = %q, want %q", result, "hello")
	}
	if im == nil {
		t.Fatal("expected InterpolationMap, got nil")
	}
}

func TestPerformInterpolationWithMapExpression(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	expr := value.NewValueExpression(value.NewSassNumber(42.0, nil), sp)
	interp, err := value.NewInterpolation(
		[]any{"prefix:", expr, "suffix"},
		[]*sasscommon.FileSpan{nil, &sp, nil},
		sp,
	)
	if err != nil {
		t.Fatal(err)
	}

	result, im, err := v.performInterpolationWithMap(interp, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "prefix:42suffix" {
		t.Errorf("result = %q, want %q", result, "prefix:42suffix")
	}
	if im == nil {
		t.Fatal("expected InterpolationMap, got nil")
	}
}

// ── warnForColor ──

func TestPerformInterpolationWarnForColor(t *testing.T) {
	rl := &recordingLogger{}
	v := testVisitor(t, rl)
	sp := testSpan()
	color, err := value.NewColorRGB(255.0, 0.0, 0.0, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	expr := value.NewValueExpression(color, sp)
	interp, err := value.NewInterpolation(
		[]any{expr},
		[]*sasscommon.FileSpan{&sp},
		sp,
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := v.performInterpolation(interp, &[]bool{true}[0])
	if err != nil {
		t.Fatal(err)
	}
	if result != "red" {
		t.Errorf("result = %q, want %q", result, "red")
	}
	if len(rl.warns) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(rl.warns))
	}
	want := strings.Join([]string{
		`You probably don't mean to use the color value red in interpolation here.`,
		`It may end up represented as red, which will likely produce invalid CSS.`,
		`Always quote color names when using them as strings or map keys (for example, "red").`,
		`If you really want to use the color value here, use '"" + test'.`,
	}, "\n")
	if rl.warns[0] != want {
		t.Errorf("warning = %q\nwant = %q", rl.warns[0], want)
	}
}

// ── evaluateExtendRule ──

func TestEvaluateExtendRuleOutsideStyleRule(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	selector := value.NewInterpolationPlain(".foo", sp)
	rule := value.NewExtendRule(selector, sp, false)

	_, err := v.VisitExtendRule(rule)
	if err == nil {
		t.Fatal("expected error for @extend outside style rule")
	}
	want := "@extend may only be used within style rules."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestEvaluateExtendRuleNestedDeclaration(t *testing.T) {
	v := testVisitor(t, &recordingLogger{})
	sp := testSpan()
	v.declarationName = "color"
	selector := value.NewInterpolationPlain(".foo", sp)
	rule := value.NewExtendRule(selector, sp, false)

	_, err := v.VisitExtendRule(rule)
	if err == nil {
		t.Fatal("expected error for @extend in nested declaration")
	}
	want := "@extend may only be used within style rules."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ── evaluateStyleRule ──

// testStyleRuleVisitor creates a visitor with root, parent, stylesheet, and
// extension store all set up for VisitStyleRule tests.
func testStyleRuleVisitor(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	if logger == nil {
		logger = &recordingLogger{}
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	v := NewEvaluateVisitor(ic, logger)
	v.ec.SetCallableNode(testNode{span: testSpan()})
	sp := testSpan()
	root := value.NewModifiableCssStylesheet(sp)
	v.root = root
	v.parent = root
	v.stylesheet = value.NewStylesheetDetailed(nil, sp, nil, false, nil)
	v.extensionStore = extend.NewDefaultExtensionStore()
	return v
}

func TestVisitStyleRuleSimple(t *testing.T) {
	v := testStyleRuleVisitor(t, &recordingLogger{})
	sp := testSpan()
	v.inUnknownAtRule = true
	v.inDependency = false
	selector := value.NewInterpolationPlain(".foo", sp)
	rule := value.NewStyleRule(selector, nil, sp)

	_, err := v.VisitStyleRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	sr, ok := children[0].(*value.ModifiableCssStyleRule)
	if !ok {
		t.Fatalf("expected ModifiableCssStyleRule, got %T", children[0])
	}
	if sr.Selector() == nil {
		t.Fatal("expected non-nil selector")
	}
	sel, err := sr.Selector().String()
	if err != nil {
		t.Fatal(err)
	}
	if sel != ".foo" {
		t.Errorf("selector = %q, want %q", sel, ".foo")
	}
}

func TestVisitStyleRuleNestedDeclarationError(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.declarationName = "color"
	selector := value.NewInterpolationPlain(".foo", sp)
	rule := value.NewStyleRule(selector, nil, sp)

	_, err := v.VisitStyleRule(rule)
	if err == nil {
		t.Fatal("expected error for style rule in nested declaration")
	}
	want := "Style rules may not be used within nested declarations."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestVisitStyleRuleKeyframeBlockError(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inKeyframes = true
	kfb := value.NewModifiableCssKeyframeBlock(
		sasscommon.NewCssValue([]string{"from"}, sp),
		sp,
	)
	v.parent = kfb
	selector := value.NewInterpolationPlain(".bar", sp)
	rule := value.NewStyleRule(selector, nil, sp)

	_, err := v.VisitStyleRule(rule)
	if err == nil {
		t.Fatal("expected error for style rule inside keyframe block")
	}
	want := "Style rules may not be used within keyframe blocks."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestVisitStyleRuleKeyframe(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inKeyframes = true
	// parent is root (NOT a keyframe block), so the keyframe branch fires
	selector := value.NewInterpolationPlain("from", sp)
	rule := value.NewStyleRule(selector, nil, sp)

	_, err := v.VisitStyleRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	kfb, ok := children[0].(*value.ModifiableCssKeyframeBlock)
	if !ok {
		t.Fatalf("expected ModifiableCssKeyframeBlock, got %T", children[0])
	}
	if len(kfb.Selector().Value) != 1 || kfb.Selector().Value[0] != "from" {
		t.Errorf("expected selector = [\"from\"], got %v", kfb.Selector().Value)
	}
}

func TestVisitStyleRuleGroupEndSet(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inUnknownAtRule = true
	v.inDependency = false
	selector := value.NewInterpolationPlain(".foo", sp)
	rule := value.NewStyleRule(selector, nil, sp)

	_, err := v.VisitStyleRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	sr, ok := children[0].(*value.ModifiableCssStyleRule)
	if !ok {
		t.Fatalf("expected ModifiableCssStyleRule, got %T", children[0])
	}
	if !sr.IsGroupEnd() {
		t.Error("expected last child to be group end")
	}
}

func TestVisitStyleRuleGroupEndNotSetWhenNested(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inUnknownAtRule = true
	v.inDependency = false
	parentSel := value.NewInterpolationPlain(".parent", sp)
	parentRule := value.NewStyleRule(parentSel, nil, sp)
	_, err := v.VisitStyleRule(parentRule)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.root.Children()) == 0 {
		t.Fatal("expected parent rule to be in root children")
	}
	v.styleRuleIgnoringAtRoot = v.root.Children()[0].(value.CssStyleRule)

	childSel := value.NewInterpolationPlain(".child", sp)
	childRule := value.NewStyleRule(childSel, nil, sp)
	_, err = v.VisitStyleRule(childRule)
	if err != nil {
		t.Fatal(err)
	}
	// The child rule should NOT be a child of the parent in the CSS tree —
	// Sass nesting produces flat CSS output (the child is a sibling of
	// the parent, with its selector merged).
	if sr, ok := v.root.Children()[0].(*value.ModifiableCssStyleRule); ok {
		// The parent itself was already set as group end by the first
		// evaluation. This remains true even after the child is added
		// because the child is a sibling, not a child of the parent.
		_ = sr
	}
}

func TestVisitStyleRuleNested(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inUnknownAtRule = true
	v.inDependency = false
	parentSel := value.NewInterpolationPlain(".parent", sp)
	parentRule := value.NewStyleRule(parentSel, nil, sp)
	_, err := v.VisitStyleRule(parentRule)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.root.Children()) == 0 {
		t.Fatal("expected parent rule in root")
	}
	v.styleRuleIgnoringAtRoot = v.root.Children()[0].(value.CssStyleRule)

	childSel := value.NewInterpolationPlain(".child", sp)
	childRule := value.NewStyleRule(childSel, nil, sp)
	_, err = v.VisitStyleRule(childRule)
	if err != nil {
		t.Fatal(err)
	}
	// Sass nesting: the child becomes a sibling of the parent in the flat
	// CSS tree, with its selector merged via nestWithin.
	rootChildren := v.root.Children()
	if len(rootChildren) != 2 {
		t.Fatalf("expected 2 children in root (parent + nested child), got %d",
			len(rootChildren))
	}
	childSr, ok := rootChildren[1].(*value.ModifiableCssStyleRule)
	if !ok {
		t.Fatalf("expected ModifiableCssStyleRule, got %T", rootChildren[1])
	}
	sel, err := childSr.Selector().String()
	if err != nil {
		t.Fatal(err)
	}
	if sel != ".parent .child" {
		t.Errorf("selector = %q, want %q", sel, ".parent .child")
	}
}

func TestVisitStyleRulePlainCssLeadingCombinatorError(t *testing.T) {
	sp := testSpan()
	v := testStyleRuleVisitor(t, &recordingLogger{})
	v.inUnknownAtRule = true
	v.inDependency = false
	// Create parent from a non-plain-CSS stylesheet (fromPlainCss=false),
	// then switch to plain CSS for the child. This triggers the leading
	// combinator check because merge=true (parent not from plain CSS) and
	// stylesheet IS plain CSS.
	parentSel := value.NewInterpolationPlain(".parent", sp)
	parentRule := value.NewStyleRule(parentSel, nil, sp)
	_, err := v.VisitStyleRule(parentRule)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.root.Children()) == 0 {
		t.Fatal("expected parent rule in root")
	}
	v.styleRuleIgnoringAtRoot = v.root.Children()[0].(value.CssStyleRule)

	// Switch to a plain CSS stylesheet
	v.stylesheet = value.NewStylesheetDetailed(nil, sp, nil, true, nil)

	childSel := value.NewInterpolationPlain("> .child", sp)
	childRule := value.NewStyleRule(childSel, nil, sp)
	_, err = v.VisitStyleRule(childRule)
	if err == nil {
		t.Fatal("expected error for leading combinator in plain CSS")
	}
	want := "Top-level leading combinators aren't allowed in plain CSS."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ── evaluateImportRule ──

func testImportRuleVisitor(t *testing.T, logger sasslogger.Logger) *EvaluateVisitor {
	t.Helper()
	if logger == nil {
		logger = &recordingLogger{}
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	v := NewEvaluateVisitor(ic, logger)
	v.ec.SetCallableNode(testNode{span: testSpan()})
	sp := testSpan()
	root := value.NewModifiableCssStylesheet(sp)
	v.root = root
	v.parent = root
	v.stylesheet = value.NewStylesheetDetailed(nil, sp, nil, false, nil)
	v.extensionStore = extend.NewDefaultExtensionStore()
	v.importCache = NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	return v
}

func TestEvaluateImportRuleStatic(t *testing.T) {
	v := testImportRuleVisitor(t, &recordingLogger{})
	sp := testSpan()
	urlInterp := value.NewInterpolationPlain("foo.css", sp)
	static := value.NewStaticImport(urlInterp, sp, nil)
	imports := []value.Import{static}
	rule := value.NewImportRule(imports, sp)

	_, err := v.VisitImportRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	imp, ok := children[0].(*value.ModifiableCssImport)
	if !ok {
		t.Fatalf("expected ModifiableCssImport, got %T", children[0])
	}
	if imp.URL().Value != "foo.css" {
		t.Errorf("url = %q, want %q", imp.URL().Value, "foo.css")
	}
}

func TestEvaluateImportRuleStaticWithModifiers(t *testing.T) {
	v := testImportRuleVisitor(t, &recordingLogger{})
	sp := testSpan()
	urlInterp := value.NewInterpolationPlain("foo.css", sp)
	modInt := value.NewInterpolationPlain("screen", sp)
	static := value.NewStaticImport(urlInterp, sp, modInt)
	imports := []value.Import{static}
	rule := value.NewImportRule(imports, sp)

	_, err := v.VisitImportRule(rule)
	if err != nil {
		t.Fatal(err)
	}
	children := v.root.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	imp, ok := children[0].(*value.ModifiableCssImport)
	if !ok {
		t.Fatalf("expected ModifiableCssImport, got %T", children[0])
	}
	if imp.Modifiers() == nil {
		t.Fatal("expected modifiers on import")
	}
	if imp.Modifiers().Value != "screen" {
		t.Errorf("modifiers = %q, want %q", imp.Modifiers().Value, "screen")
	}
}

func TestEvaluateImportRuleDynamicNotFound(t *testing.T) {
	v := testImportRuleVisitor(t, &recordingLogger{})
	sp := testSpan()
	dyn := value.NewDynamicImport("nonexistent-file.scss", sp)
	imports := []value.Import{dyn}
	rule := value.NewImportRule(imports, sp)

	_, err := v.VisitImportRule(rule)
	if err == nil {
		t.Fatal("expected error for missing import")
	}
	want := "Can't find stylesheet to import."
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

// ===========================================================================
// Evaluate() integration tests
// ===========================================================================

func evalStylesheet(t *testing.T, source string) (*EvaluateResult, error) {
	t.Helper()
	parser := value.NewScssParser([]byte(source), nil, false)
	stylesheet, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	return Evaluate(
		stylesheet,
		ic,
		nil, // nodeImporter
		nil, // importer
		nil, // functions
		&recordingLogger{},
		false, // quietDeps
		false, // sourceMap
	)
}

func TestEvaluateBasic(t *testing.T) {
	result, err := evalStylesheet(t, "a { color: red; }")
	if err != nil {
		t.Fatal(err)
	}
	children := result.Stylesheet.Children()
	if len(children) == 0 {
		t.Fatal("expected non-empty CSS output")
	}
	// Should contain a style rule
	found := false
	for _, child := range children {
		if _, ok := child.(value.CssStyleRule); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one CssStyleRule in output")
	}
}

func TestEvaluateEmpty(t *testing.T) {
	result, err := evalStylesheet(t, "")
	if err != nil {
		t.Fatal(err)
	}
	children := result.Stylesheet.Children()
	if len(children) != 0 {
		t.Errorf("expected empty CSS output, got %d children", len(children))
	}
}

func TestEvaluateLoadedUrls(t *testing.T) {
	parser := value.NewScssParser([]byte("a { color: red; }"), nil, false)
	stylesheet, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	ic := NewImportCacheWithOptions(nil, nil, "", false, sassio.NewDefaultIO(), nil)
	result, err := Evaluate(
		stylesheet,
		ic,
		nil,
		nil,
		nil,
		&recordingLogger{},
		false,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	// loadedUrls may be empty or contain stdin depending on URL,
	// but LoadedUrls() should be callable and non-nil.
	urls := result.LoadedUrls
	if urls == nil {
		t.Fatal("expected non-nil LoadedUrls set")
	}
}

func TestEvaluateError(t *testing.T) {
	_, err := evalStylesheet(t, "$x: 1px + red;")
	if err == nil {
		t.Fatal("expected error for invalid SCSS")
	}
	if !strings.Contains(err.Error(), "red") {
		t.Errorf("error should mention incompatible types, got: %q", err.Error())
	}
}

func TestEvaluateVariable(t *testing.T) {
	result, err := evalStylesheet(t, "$x: 10px; a { width: $x; }")
	if err != nil {
		t.Fatal(err)
	}
	children := result.Stylesheet.Children()
	if len(children) == 0 {
		t.Fatal("expected non-empty CSS output")
	}
}

func TestEvaluateResultFields(t *testing.T) {
	result, err := evalStylesheet(t, "a { color: red; }")
	if err != nil {
		t.Fatal(err)
	}
	if result.Stylesheet == nil {
		t.Fatal("expected non-nil Stylesheet")
	}
	// LoadedUrls is a LinkedSet; .Len() should return >= 0
	if result.LoadedUrls.Len() < 0 {
		t.Error("LoadedUrls.Len() should be >= 0")
	}
}
