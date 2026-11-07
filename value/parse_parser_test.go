package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/util"
)

func newTestParser(text string) *Parser {
	return NewParser([]byte(text), nil, nil)
}

// --- Static helpers ---

func TestParseIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"foo", "foo", true},
		{"foo-bar", "foo-bar", true},
		{"café", "café", true},
		{"--custom-prop", "--custom-prop", true},
		{"_private", "_private", true},
		{`\41`, "A", true},
		{"", "", false},
		{"1invalid", "1invalid", false},
	}

	for _, tt := range tests {
		got, err := ParseIdentifier(tt.input)
		if tt.ok && err != nil {
			t.Errorf("ParseIdentifier(%q): unexpected error: %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("ParseIdentifier(%q): expected error, got %q", tt.input, got)
		}
		if tt.ok && got != tt.want {
			t.Errorf("ParseIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsIdentifier(t *testing.T) {
	if !IsIdentifier("foo") {
		t.Error("IsIdentifier(\"foo\") should be true")
	}
	if IsIdentifier("1foo") {
		t.Error("IsIdentifier(\"1foo\") should be false")
	}
	if IsIdentifier("") {
		t.Error("IsIdentifier(\"\") should be false")
	}
}

func TestIsVariableDeclarationLike(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"$var: val", true},
		{"$var : val", true},
		{"$var val", false},
		{"$1var: val", false},
		{"x: 1", false},
		{"", false},
		{"$: val", false},
	}

	for _, tt := range tests {
		got, err := IsVariableDeclarationLike(tt.input)
		if err != nil {
			t.Errorf("IsVariableDeclarationLike(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("IsVariableDeclarationLike(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Identifier methods ---

func TestIdentifierPlain(t *testing.T) {
	p := newTestParser("foo")
	got, err := p.identifier(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("identifier = %q, want %q", got, "foo")
	}
}

func TestIdentifierNormalized(t *testing.T) {
	p := newTestParser("_bar")
	got, err := p.identifier(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "-bar" {
		t.Errorf("identifier (normalized) = %q, want %q", got, "-bar")
	}
}

func TestIdentifierUnderscoreBodyNormalized(t *testing.T) {
	p := newTestParser("foo_bar")
	got, err := p.identifier(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo-bar" {
		t.Errorf("identifier (normalized) = %q, want %q", got, "foo-bar")
	}
}

func TestIdentifierDoubleDash(t *testing.T) {
	p := newTestParser("--my-prop")
	got, err := p.identifier(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "--my-prop" {
		t.Errorf("identifier = %q, want %q", got, "--my-prop")
	}
}

func TestIdentifierUnitMode(t *testing.T) {
	// unit mode: dash after start is allowed only when next is not digit or '.'
	// "foo-bar" — dash followed by 'b' (identifier body), allowed
	p := newTestParser("foo-bar")
	got, err := p.identifier(false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo-bar" {
		t.Errorf("identifier (unit) = %q, want %q", got, "foo-bar")
	}
}

func TestIdentifierUnitModeDashBeforeDigit(t *testing.T) {
	// In unit mode, a dash in the body before a digit returns (stops at dash)
	p := newTestParser("foo-5")
	got, err := p.identifier(false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("identifier (unit) = %q, want %q", got, "foo")
	}
}

func TestIdentifierEscape(t *testing.T) {
	// \41 is ASCII 'A'
	p := newTestParser(`\41 bc`)
	got, err := p.identifier(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Abc" {
		t.Errorf("identifier = %q, want %q", got, "Abc")
	}
}

func TestIdentifierBody(t *testing.T) {
	p := newTestParser("foo")
	p.scanner.ScanChar('f')
	got, err := p.identifierBody()
	if err != nil {
		t.Fatal(err)
	}
	if got != "oo" {
		t.Errorf("identifierBody = %q, want %q", got, "oo")
	}
}

func TestIdentifierBodyEmpty(t *testing.T) {
	p := newTestParser(".")
	p.scanner.ScanChar('.')
	_, err := p.identifierBody()
	if err == nil {
		t.Error("identifierBody: expected error for empty body")
	}
	want := "Expected identifier body."
	if err.Error() != want {
		t.Errorf("identifierBody error = %q, want %q", err.Error(), want)
	}
}

// --- String ---

func TestStringSingleQuote(t *testing.T) {
	p := newTestParser("'hello'")
	got, err := p.string()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("string = %q, want %q", got, "hello")
	}
}

func TestStringDoubleQuote(t *testing.T) {
	p := newTestParser(`"world"`)
	got, err := p.string()
	if err != nil {
		t.Fatal(err)
	}
	if got != "world" {
		t.Errorf("string = %q, want %q", got, "world")
	}
}

func TestStringEscape(t *testing.T) {
	p := newTestParser(`"\41"`)
	got, err := p.string()
	if err != nil {
		t.Fatal(err)
	}
	if got != "A" {
		t.Errorf("string = %q, want %q", got, "A")
	}
}

func TestStringEscapeNewline(t *testing.T) {
	p := newTestParser("\"a\\\nb\"")
	got, err := p.string()
	if err != nil {
		t.Fatal(err)
	}
	if got != "ab" {
		t.Errorf("string = %q, want %q", got, "ab")
	}
}

// --- Numbers ---

func TestNaturalNumber(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"42", 42},
		{"0", 0},
		{"999", 999},
	}
	for _, tt := range tests {
		p := newTestParser(tt.input)
		got, err := p.naturalNumber()
		if err != nil {
			t.Errorf("naturalNumber(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("naturalNumber(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestNaturalNumberError(t *testing.T) {
	p := newTestParser("abc")
	_, err := p.naturalNumber()
	if err == nil {
		t.Error("naturalNumber: expected error for non-digit")
	}
	if err.Error() != "Expected digit." {
		t.Errorf("naturalNumber error = %q, want %q", err.Error(), "Expected digit.")
	}
}

// --- Variable name ---

func TestVariableName(t *testing.T) {
	p := newTestParser("$foo-bar")
	got, err := p.variableName()
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo-bar" {
		t.Errorf("variableName = %q, want %q", got, "foo-bar")
	}
}

func TestVariableNameError(t *testing.T) {
	p := newTestParser("foo")
	_, err := p.variableName()
	if err == nil {
		t.Error("variableName: expected error for missing $")
	}
	if err.Error() != "expected \"$\"." {
		t.Errorf("variableName error = %q, want %q", err.Error(), "expected \"$\".")
	}
}

// --- Escape sequences ---

func TestEscapeResolvesCharacters(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`\41`, 'A'},
		{`\41 `, 'A'},
		{`\1`, 1},
		{`\@`, '@'},
		{`\g`, 'g'},
		{`\0`, 0xFFFD},
		{`\D800 `, 0xFFFD},
		{`\000041`, 'A'},
	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got, err := p.escape()
		if err != nil {
			t.Errorf("escape(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("escape(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestEscapeEndOfInput(t *testing.T) {
	p := newTestParser(`\`)
	got, err := p.escape()
	if err != nil {
		t.Fatal(err)
	}
	if got != 0xFFFD {
		t.Errorf("escape at EOF = %d, want 0xFFFD", got)
	}
}

func TestEscapeNewlineError(t *testing.T) {
	p := newTestParser("\\\n")
	_, err := p.escape()
	if err == nil {
		t.Error("escape: expected error for newline after \\")
	}
	if err.Error() != "Expected escape sequence." {
		t.Errorf("escape error = %q, want %q", err.Error(), "Expected escape sequence.")
	}
}

func TestCharacterToEscape(t *testing.T) {
	tests := []struct {
		character       int
		identifierStart bool
		want            string
	}{
		{'A', true, "A"},
		{'1', true, `\31 `},
		{'1', false, "1"},
		{0x01, true, `\1 `},
		{0x7F, false, `\7f `},
		{'@', true, `\@`},
	}

	for _, tt := range tests {
		if got := util.CharacterToEscape(tt.character, tt.identifierStart); got != tt.want {
			t.Errorf("CharacterToEscape(%d, %v) = %q, want %q", tt.character, tt.identifierStart, got, tt.want)
		}
	}
}

// ==========================================================================
// Scan ident char case-insensitive + escape error
// ==========================================================================

func TestScanIdentCharCaseInsensitiveMatch(t *testing.T) {
	p := newTestParser("Abc")
	ok, err := p.scanIdentChar('a', false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanIdentChar: case-insensitive should match 'A' with 'a'")
	}
}

func TestScanIdentCharEscapeError(t *testing.T) {
	p := newTestParser("\\\n")
	// \ followed by newline is an escape error
	_, err := p.scanIdentChar('a', false)
	if err == nil {
		t.Error("scanIdentChar: expected error for invalid escape")
	}
}

// ==========================================================================
// Lookahead edge cases
// ==========================================================================

func TestLookingAtNumberPlusAlone(t *testing.T) {
	p := newTestParser("+")
	if p.lookingAtNumber() {
		t.Error("lookingAtNumber: '+' alone should be false")
	}
}

func TestLookingAtNumberMinusAlone(t *testing.T) {
	p := newTestParser("-")
	if p.lookingAtNumber() {
		t.Error("lookingAtNumber: '-' alone should be false")
	}
}

func TestLookingAtNumberBadDot(t *testing.T) {
	p := newTestParser("+.abc")
	if p.lookingAtNumber() {
		t.Error("lookingAtNumber: '+.abc' should be false")
	}
}

func TestLookingAtIdentifierDoubleDash(t *testing.T) {
	p := newTestParser("--a")
	f := 0
	if !p.lookingAtIdentifier(&f) {
		t.Error("lookingAtIdentifier: '--a' should match (double dash start)")
	}
}

func TestLookingAtIdentifierEscape(t *testing.T) {
	p := newTestParser(`\a`)
	if !p.lookingAtIdentifier(nil) {
		t.Error("lookingAtIdentifier: '\\a' should match")
	}
}

func TestLookingAtIdentifierForwardDash(t *testing.T) {
	p := newTestParser("abc-def")
	f := 3 // position of '-'
	if !p.lookingAtIdentifier(&f) {
		t.Error("lookingAtIdentifier forward: should see '-' then 'def' as identifier")
	}
}

func TestLookingAtIdentifierBodyDash(t *testing.T) {
	p := newTestParser("-")
	if !p.lookingAtIdentifierBody() {
		t.Error("lookingAtIdentifierBody: '-' should match")
	}
}

func TestLookingAtIdentifierBodyNonName(t *testing.T) {
	p := newTestParser("{")
	if p.lookingAtIdentifierBody() {
		t.Error("lookingAtIdentifierBody: '{' should not match")
	}
}

// ==========================================================================
// Identifier scanning edge cases
// ==========================================================================

func TestScanIdentifierCaseInsensitive(t *testing.T) {
	p := newTestParser("Foo")
	ok, err := p.scanIdentifier("foo", false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanIdentifier: case-insensitive should match 'Foo' with 'foo'")
	}
	if p.scanner.Position() != 3 {
		t.Errorf("scanIdentifier: pos = %d, want 3", p.scanner.Position())
	}
}

func TestScanIdentifierNotIdentifier(t *testing.T) {
	p := newTestParser("!foo")
	ok, err := p.scanIdentifier("foo", true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanIdentifier: '!foo' should not match (not looking at identifier)")
	}
	if p.scanner.Position() != 0 {
		t.Errorf("scanIdentifier: pos = %d, want 0 (unchanged)", p.scanner.Position())
	}
}

func TestExpectIdentifierTrailingBody(t *testing.T) {
	p := newTestParser("foo123")
	err := p.expectIdentifier("foo", "", true)
	if err == nil {
		t.Error("expectIdentifier: expected error for trailing body chars")
	}
	if err.Error() != "Expected \"foo\"" {
		t.Errorf("expectIdentifier error = %q, want %q", err.Error(), "Expected \"foo\"")
	}
}

func TestExpectIdentifierCustomName(t *testing.T) {
	// Test with a custom name parameter
	p := newTestParser("foo")
	err := p.expectIdentifier("bar", "a name", true)
	if err == nil {
		t.Error("expectIdentifier: expected error")
	}
	if err.Error() != "Expected a name." {
		t.Errorf("expectIdentifier error = %q, want %q", err.Error(), "Expected a name.")
	}
}

// ==========================================================================
// Span utilities
// ==========================================================================

func TestSpanFrom(t *testing.T) {
	p := newTestParser("hello world")
	start := p.scanner.State()
	for i := 0; i < 5; i++ {
		p.readChar()
	}
	span, err := p.spanFrom(start)
	if err != nil {
		t.Fatal(err)
	}
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("spanFrom text = %q, want %q", text, "hello")
	}
}

func TestSpanFromTo(t *testing.T) {
	p := newTestParser("foobar")
	s1 := p.scanner.State()
	p.readChar()
	p.readChar()
	p.readChar()
	s2 := p.scanner.State()
	span, err := p.spanFromTo(s1, &s2)
	if err != nil {
		t.Fatal(err)
	}
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "foo" {
		t.Errorf("spanFromTo text = %q, want %q", text, "foo")
	}
}

func TestSpanFromPosition(t *testing.T) {
	p := newTestParser("hello")
	for i := 0; i < 5; i++ {
		p.readChar()
	}
	span, err := p.spanFromPosition(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Errorf("spanFromPosition text = %q, want %q", text, "hello")
	}
}

func TestSpanFromPositionEnd(t *testing.T) {
	p := newTestParser("hello")
	for i := 0; i < 5; i++ {
		p.readChar()
	}
	end := 2
	span, err := p.spanFromPosition(0, &end)
	if err != nil {
		t.Fatal(err)
	}
	text, err := span.SpanText()
	if err != nil {
		t.Fatal(err)
	}
	if text != "he" {
		t.Errorf("spanFromPosition text = %q, want %q", text, "he")
	}
}

// ==========================================================================
// Error methods
// ==========================================================================

func TestErrorImpl(t *testing.T) {
	p := newTestParser("x")
	span := p.scanner.EmptySpan()
	err := p.error("test message", span, nil)
	if err == nil {
		t.Error("error: expected non-nil error")
	}
	want := strings.Join([]string{
		"Error: test message",
		"  ╷",
		"1 │ x",
		"  │ ^",
		"  ╵",
		"  - 1:1  root stylesheet",
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestMultiSpanErrorImpl(t *testing.T) {
	p := newTestParser("x")
	span := p.scanner.EmptySpan()
	secondary := sasscommon.NewSecondarySpans(sasscommon.SecondarySpan{Span: span, Label: "secondary desc"})
	err := p.multiSpanError("main msg", span, "primary label", secondary)
	if err == nil {
		t.Error("multiSpanError: expected non-nil error")
	}
	want := strings.Join([]string{
		"Error: main msg",
		"  ╷",
		"1 │ x",
		"  │ ^ primary label",
		"  ╵",
		"  ╷",
		"1 │ x",
		"  │ ━ secondary desc",
		"  ╵",
		"  - 1:1  root stylesheet",
	}, "\n")
	if err.Error() != want {
		t.Errorf("multiSpanError = %q, want %q", err.Error(), want)
	}
}

// ==========================================================================
// Consume escaped character edge cases
// ==========================================================================

func TestConsumeEscapedCharacterEof(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0xFFFD {
		t.Errorf("consumeEscapedCharacter = %d, want 0xFFFD", got)
	}
}

func TestConsumeEscapedCharacterOverflow(t *testing.T) {
	// \FFFFFF is 6 hex digits = 0xFFFFFF which exceeds MAX_ALLOWED_CHARACTER
	scanner := sasscommon.NewSpanScanner([]byte(`\FFFFFF`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0xFFFD {
		t.Errorf("consumeEscapedCharacter = %d, want 0xFFFD", got)
	}
}

func TestConsumeEscapedCharacterPlain(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\!`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != '!' {
		t.Errorf("consumeEscapedCharacter = %d, want %d", got, '!')
	}
}

func TestConsumeEscapedCharacterHexZero(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\0 `), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0xFFFD {
		t.Errorf("consumeEscapedCharacter(\\0) = %d, want 0xFFFD", got)
	}
}

func TestResolveCssEscapes(t *testing.T) {
	got, err := ResolveCssEscapes(`f\6fo`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("ResolveCssEscapes = %q, want %q", got, "foo")
	}

	_, err = ResolveCssEscapes("*x")
	if err == nil {
		t.Fatal("ResolveCssEscapes(*x): expected error")
	}
	argErr, ok := err.(*sasscommon.ArgumentError)
	if !ok {
		t.Fatalf("error = %T, want *sasscommon.ArgumentError", err)
	}
	want := `"*x" isn't a valid CSS identifier`
	if argErr.Message != want {
		t.Errorf("error = %q, want %q", argErr.Message, want)
	}
}
