package value

import (
	"strings"
	"testing"

	"github.com/bancek/go-sass/sasscommon"
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

func TestEscapeHex(t *testing.T) {
	p := newTestParser(`\41`)
	got, err := p.escape(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "A" {
		t.Errorf("escape = %q, want %q", got, "A")
	}
}

func TestEscapeHexTrailingSpace(t *testing.T) {
	p := newTestParser(`\41 `)
	got, err := p.escape(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "A" {
		t.Errorf("escape = %q, want %q", got, "A")
	}
}

func TestEscapeControlChar(t *testing.T) {
	// \1 is control char, should return \1[space]
	p := newTestParser(`\1`)
	got, err := p.escape(false)
	if err != nil {
		t.Fatal(err)
	}
	want := `\1 `
	if got != want {
		t.Errorf("escape control: got %q, want %q", got, want)
	}
}

func TestEscapeNonHex(t *testing.T) {
	// \@ returns \@ (backslash preserved for non-name char)
	p := newTestParser(`\@`)
	got, err := p.escape(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != `\@` {
		t.Errorf("escape = %q, want %q", got, `\@`)
	}
}

func TestEscapeNewlineError(t *testing.T) {
	p := newTestParser("\\\n")
	_, err := p.escape(false)
	if err == nil {
		t.Error("escape: expected error for newline after \\")
	}
	if err.Error() != "Expected escape sequence." {
		t.Errorf("escape error = %q, want %q", err.Error(), "Expected escape sequence.")
	}
}

// --- Escape character ---

func TestEscapeCharacter(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`\41`, 65},    // 'A'
		{`\a`, 10},     // line feed
		{`\0`, 0xFFFD}, // null → replacement char
		{`\*`, '*'},    // plain char
		{`\9`, 9},      // tab
	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got, err := p.escapeCharacter()
		if err != nil {
			t.Errorf("escapeCharacter(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("escapeCharacter(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

// --- Character matching ---

func TestScanCharIf(t *testing.T) {
	p := newTestParser("42")
	ok, err := p.scanCharIf(func(ch int) bool { return ch >= '0' && ch <= '9' })
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanCharIf: should match digit")
	}
	if p.scanner.Position() != 1 {
		t.Errorf("scanCharIf: pos = %d, want 1", p.scanner.Position())
	}
}

func TestScanCharIfNoMatch(t *testing.T) {
	p := newTestParser("abc")
	ok, err := p.scanCharIf(func(ch int) bool { return ch >= '0' && ch <= '9' })
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanCharIf: should not match")
	}
	if p.scanner.Position() != 0 {
		t.Errorf("scanCharIf: pos = %d, want 0 (no advance)", p.scanner.Position())
	}
}

func TestScanIdentCharExact(t *testing.T) {
	p := newTestParser("abc")
	ok, err := p.scanIdentChar('a', true)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanIdentChar: should match 'a' exactly")
	}
}

func TestScanIdentCharCaseSensitiveNoMatch(t *testing.T) {
	p := newTestParser("Abc")
	ok, err := p.scanIdentChar('a', true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanIdentChar: should not match 'a' vs 'A' case-sensitively")
	}
}

func TestScanIdentCharCaseInsensitive(t *testing.T) {
	p := newTestParser("Abc")
	ok, err := p.scanIdentChar('a', false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanIdentChar: should match 'a' vs 'A' case-insensitively")
	}
}

func TestScanIdentCharEscapeMatch(t *testing.T) {
	// \41 = 'A', case-insensitive match with 'a'
	p := newTestParser(`\41`)
	ok, err := p.scanIdentChar('a', false)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanIdentChar: should match escape \\41 with 'a'")
	}
}

func TestScanIdentCharEscapeNoMatch(t *testing.T) {
	// \42 = 'B', case-insensitive match with 'a' should fail
	p := newTestParser(`\42`)
	ok, err := p.scanIdentChar('a', false)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanIdentChar: should not match \\42 with 'a'")
	}
	// Position should be restored
	if p.scanner.Position() != 0 {
		t.Errorf("scanIdentChar: pos = %d, want 0 (restored)", p.scanner.Position())
	}
}

func TestExpectIdentChar(t *testing.T) {
	p := newTestParser("abc")
	err := p.expectIdentChar('a', true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExpectIdentCharError(t *testing.T) {
	p := newTestParser("abc")
	err := p.expectIdentChar('x', true)
	if err == nil {
		t.Error("expectIdentChar: expected error")
	}
	if err.Error() != "Expected \"x\"." {
		t.Errorf("expectIdentChar error = %q, want %q", err.Error(), "Expected \"x\".")
	}
}

// --- Lookahead predicates ---

func TestLookingAtNumber(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1", true},
		{".5", true},
		{"+.5", true},
		{"-1", true},
		{"-.5", true},
		{"a", false},
		{".", false},
		{"+", false},
		{"-", false},
		{"", false},
	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got := p.lookingAtNumber()
		if got != tt.want {
			t.Errorf("lookingAtNumber(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLookingAtIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"_a", true},
		{"-a", true},
		{"--a", true},
		{`\a`, true},
		{"1a", false},
		{"", false},
		{".", false},
	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got := p.lookingAtIdentifier(nil)
		if got != tt.want {
			t.Errorf("lookingAtIdentifier(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLookingAtIdentifierBody(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"a", true},
		{"-", true},
		{`\a`, true},
		{".", false},
		{"", false},
		{"{", false},
	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got := p.lookingAtIdentifierBody()
		if got != tt.want {
			t.Errorf("lookingAtIdentifierBody(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Identifier scanning ---

func TestScanIdentifier(t *testing.T) {
	tests := []struct {
		input string
		text  string
		want  bool
	}{
		{"foo", "foo", true},
		{"foobar", "foo", false}, // trailing chars
		{"Foo", "foo", false},    // case-sensitive: 'Foo' ≠ 'foo'
		{"bar", "foo", false},    // no match
		{"foo", "f", false},      // partial: "f" matches but "oo" is identifier body

	}

	for _, tt := range tests {
		p := newTestParser(tt.input)
		got, err := p.scanIdentifier(tt.text, true)
		if err != nil {
			t.Errorf("scanIdentifier(%q, %q): %v", tt.input, tt.text, err)
			continue
		}
		if got != tt.want {
			t.Errorf("scanIdentifier(%q, %q) = %v, want %v", tt.input, tt.text, got, tt.want)
		}
	}
}

func TestMatchesIdentifier(t *testing.T) {
	p := newTestParser("foobar")
	if p.matchesIdentifier("foo", true) {
		t.Error("matchesIdentifier: 'foo' should not match 'foobar'")
	}
	// Verify position is unchanged
	if p.scanner.Position() != 0 {
		t.Errorf("matchesIdentifier: pos = %d, want 0 (unchanged)", p.scanner.Position())
	}
}

func TestMatchesIdentifierTrue(t *testing.T) {
	p := newTestParser("foo bar")
	if !p.matchesIdentifier("foo", true) {
		t.Error("matchesIdentifier: 'foo' should match 'foo'")
	}
	if p.scanner.Position() != 0 {
		t.Errorf("matchesIdentifier: pos = %d, want 0 (unchanged)", p.scanner.Position())
	}
}

func TestExpectIdentifierSuccess(t *testing.T) {
	p := newTestParser("foo bar")
	err := p.expectIdentifier("foo", "a name", true)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 3 {
		t.Errorf("expectIdentifier: pos = %d, want 3", p.scanner.Position())
	}
}

func TestExpectIdentifierError(t *testing.T) {
	p := newTestParser("foo")
	err := p.expectIdentifier("bar", "a name", true)
	if err == nil {
		t.Error("expectIdentifier: expected error")
	}
}

// --- raw_text ---

func TestRawText(t *testing.T) {
	p := newTestParser("hello world")
	got, err := p.rawText(func() error {
		for i := 0; i < 5; i++ {
			_, e := p.readChar()
			if e != nil {
				return e
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("rawText = %q, want %q", got, "hello")
	}
}

// --- declaration_value ---

func TestDeclarationValueBrackets(t *testing.T) {
	p := newTestParser("(foo);")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "(foo)" {
		t.Errorf("declarationValue = %q, want %q", got, "(foo)")
	}
}

func TestDeclarationValueNestedBrackets(t *testing.T) {
	p := newTestParser("a(b(c)d);")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a(b(c)d)" {
		t.Errorf("declarationValue = %q, want %q", got, "a(b(c)d)")
	}
}

func TestDeclarationValueSemicolonTerminate(t *testing.T) {
	p := newTestParser("foo; bar")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("declarationValue = %q, want %q", got, "foo")
	}
}

func TestDeclarationValueUnmatchedCloseBracket(t *testing.T) {
	p := newTestParser("foo);")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("declarationValue = %q, want %q", got, "foo")
	}
}

func TestDeclarationValueSpaceCollapse(t *testing.T) {
	p := newTestParser("a  b;")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a b" {
		t.Errorf("declarationValue = %q, want %q", got, "a b")
	}
}

func TestDeclarationValueUrl(t *testing.T) {
	p := newTestParser("url(foo) bar;")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(foo) bar" {
		t.Errorf("declarationValue = %q, want %q", got, "url(foo) bar")
	}
}

// --- try_url ---

func TestTryUrl(t *testing.T) {
	p := newTestParser("url(foo)bar")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(foo)" {
		t.Errorf("tryUrl = %q, want %q", got, "url(foo)")
	}
}

func TestTryUrlNotUrl(t *testing.T) {
	p := newTestParser("xyz bar")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("tryUrl = %q, want empty", got)
	}
	if p.scanner.Position() != 0 {
		t.Errorf("tryUrl: pos = %d, want 0 (restored)", p.scanner.Position())
	}
}

// --- Whitespace ---

func TestWhitespace(t *testing.T) {
	p := newTestParser("  \t\nx")
	err := p.whitespace(true)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 4 {
		t.Errorf("whitespace: pos = %d, want 4", p.scanner.Position())
	}
}

func TestSpaces(t *testing.T) {
	p := newTestParser("  \tx")
	err := p.spaces()
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 3 {
		t.Errorf("spaces: pos = %d, want 3", p.scanner.Position())
	}
}

// --- Comments ---

func TestLoudComment(t *testing.T) {
	p := newTestParser("/*hello*/x")
	err := p.loudComment()
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 9 {
		t.Errorf("loudComment: pos = %d, want 9", p.scanner.Position())
	}
}

func TestSilentComment(t *testing.T) {
	p := newTestParser("// hello\nx")
	ok, err := p.silentComment()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("silentComment should return true")
	}
	if p.scanner.Position() != 8 {
		t.Errorf("silentComment: pos = %d, want 8", p.scanner.Position())
	}
}

func TestSilentCommentNotComment(t *testing.T) {
	p := newTestParser("x")
	_, err := p.silentComment()
	if err == nil {
		t.Error("silentComment: expected error for non-comment")
	}
	if err.Error() != "expected \"//\"." {
		t.Errorf("silentComment error = %q, want %q", err.Error(), "expected \"//\".")
	}
}

func TestScanCommentSlashSlash(t *testing.T) {
	p := newTestParser("//hello\nx")
	ok, err := p.scanComment()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanComment: // should be recognized")
	}
}

func TestScanCommentSlashStar(t *testing.T) {
	p := newTestParser("/*hello*/x")
	ok, err := p.scanComment()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("scanComment: /* */ should be recognized")
	}
}

func TestScanCommentNotComment(t *testing.T) {
	p := newTestParser("x")
	ok, err := p.scanComment()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanComment: should return false for non-comment")
	}
}

// --- Consume escaped character ---

func TestConsumeEscapedCharacter(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\41`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != 65 {
		t.Errorf("consumeEscapedCharacter = %d, want 65", got)
	}
}

func TestConsumeEscapedCharacterSurrogate(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\D800`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0xFFFD {
		t.Errorf("consumeEscapedCharacter = %d, want 0xFFFD", got)
	}
}

func TestConsumeEscapedCharacterPlainChar(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte(`\*`), nil)
	got, err := consumeEscapedCharacter(scanner)
	if err != nil {
		t.Fatal(err)
	}
	if got != '*' {
		t.Errorf("consumeEscapedCharacter = %d, want '*'", got)
	}
}

func TestConsumeEscapedCharacterNewlineError(t *testing.T) {
	scanner := sasscommon.NewSpanScanner([]byte("\\\n"), nil)
	_, err := consumeEscapedCharacter(scanner)
	if err == nil {
		t.Error("consumeEscapedCharacter: expected error for newline after \\")
	}
	want := "Expected escape sequence."
	if err.Error() != want {
		t.Errorf("consumeEscapedCharacter error = %q, want %q", err.Error(), want)
	}
}

// --- hasPrefixIgnoreCase / asciiCharEqualsIgnoreCase ---

func TestHasPrefixIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		prefix string
		want   bool
	}{
		{"Expected foo", "expected", true},
		{"expected", "Expected", true},
		{"Foo", "foo", true},
		{"Foo", "Foo", true},
		{"Foo", "", true},
		{"Foo", "foobar", false},
	}

	for _, tt := range tests {
		got := hasPrefixIgnoreCase(tt.s, tt.prefix)
		if got != tt.want {
			t.Errorf("hasPrefixIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
		}
	}
}

func TestAsciiCharEqualsIgnoreCase(t *testing.T) {
	if !asciiCharEqualsIgnoreCase('a', 'A') {
		t.Error("'a' should equal 'A'")
	}
	if !asciiCharEqualsIgnoreCase('Z', 'z') {
		t.Error("'Z' should equal 'z'")
	}
	if asciiCharEqualsIgnoreCase('a', 'b') {
		t.Error("'a' should not equal 'b'")
	}
	if asciiCharEqualsIgnoreCase('0', 'P') {
		t.Error("'0' should not equal 'P'")
	}
}

// --- wrapSpanFormatException ---

func TestWrapSpanFormatException(t *testing.T) {
	p := newTestParser("foo")
	err := p.wrapSpanFormatException(func() error {
		return p.scanner.Error("expected test.", 0, 0)
	})
	if err == nil {
		t.Error("wrapSpanFormatException: expected error")
	}
}

// --- lookahead at identifier edge cases ---

func TestLookingAtIdentifierForwards(t *testing.T) {
	p := newTestParser("abc-def")
	// Check at position 4: looking at '-' then identifier
	f := 3 // pointing at '-'
	got := p.lookingAtIdentifier(&f)
	if !got {
		t.Error("lookingAtIdentifier with forward: should see '-' + identifier")
	}
}

func TestScanIdentifierNoMatchRestores(t *testing.T) {
	p := newTestParser("variable")
	ok, err := p.scanIdentifier("var", true)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("scanIdentifier: 'var' should not match 'variable' (trailing body chars)")
	}
	if p.scanner.Position() != 0 {
		t.Errorf("scanIdentifier: pos = %d, want 0 (restored)", p.scanner.Position())
	}
}

// ==========================================================================
// Scanner delegates
// ==========================================================================

func TestReadChar(t *testing.T) {
	p := newTestParser("abc")
	ch, err := p.readChar()
	if err != nil {
		t.Fatal(err)
	}
	if ch != 'a' {
		t.Errorf("readChar = %c, want 'a'", ch)
	}
	if p.scanner.Position() != 1 {
		t.Errorf("readChar: pos = %d, want 1", p.scanner.Position())
	}
}

func TestReadCharEof(t *testing.T) {
	p := newTestParser("")
	_, err := p.readChar()
	if err == nil {
		t.Error("readChar: expected error at EOF")
	}
}

func TestSetPosition(t *testing.T) {
	p := newTestParser("hello")
	p.readChar()
	p.readChar()
	p.readChar()
	err := p.setPosition(0)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 0 {
		t.Errorf("setPosition: pos = %d, want 0", p.scanner.Position())
	}
}

func TestSetPositionBeyond(t *testing.T) {
	p := newTestParser("abc")
	err := p.setPosition(100)
	if err == nil {
		t.Error("setPosition: expected error for position beyond end")
	}
}

func TestExpectChar(t *testing.T) {
	p := newTestParser("$var")
	err := p.expectChar('$')
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 1 {
		t.Errorf("expectChar: pos = %d, want 1", p.scanner.Position())
	}
}

func TestExpectCharError(t *testing.T) {
	p := newTestParser("x")
	err := p.expectChar('$')
	if err == nil {
		t.Error("expectChar: expected error")
	}
	if err.Error() != "expected \"$\"." {
		t.Errorf("expectChar error = %q, want %q", err.Error(), "expected \"$\".")
	}
}

func TestExpectCharName(t *testing.T) {
	p := newTestParser("$var")
	err := p.expectCharName('$', "dollar sign")
	if err != nil {
		t.Fatal(err)
	}
}

func TestExpectCharNameError(t *testing.T) {
	p := newTestParser("x")
	err := p.expectCharName('$', "dollar sign")
	if err == nil {
		t.Error("expectCharName: expected error")
	}
	if err.Error() != "expected dollar sign." {
		t.Errorf("expectCharName error = %q, want %q", err.Error(), "expected dollar sign.")
	}
}

func TestExpect(t *testing.T) {
	p := newTestParser("/*hello*/")
	err := p.expect("/*")
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 2 {
		t.Errorf("expect: pos = %d, want 2", p.scanner.Position())
	}
}

func TestExpectError(t *testing.T) {
	p := newTestParser("x")
	err := p.expect("//")
	if err == nil {
		t.Error("expect: expected error")
	}
	if err.Error() != "expected \"//\"." {
		t.Errorf("expect error = %q, want %q", err.Error(), "expected \"//\".")
	}
}

// ==========================================================================
// Whitespace without comments (syntax modes)
// ==========================================================================

func TestWhitespaceWithoutCommentsScss(t *testing.T) {
	p := newTestParser("  \t  x")
	err := p.whitespaceWithoutComments(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 5 {
		t.Errorf("pos = %d, want 5", p.scanner.Position())
	}
}

func TestWhitespaceWithoutCommentsNoWs(t *testing.T) {
	p := newTestParser("x")
	err := p.whitespaceWithoutComments(true)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 0 {
		t.Errorf("pos = %d, want 0 (unchanged)", p.scanner.Position())
	}
}

// ==========================================================================
// Comments — edge cases (CSS tests require StylesheetParser, tested there)
// ==========================================================================

// ==========================================================================
// Loud comment edge cases
// ==========================================================================

func TestLoudCommentNestedStar(t *testing.T) {
	p := newTestParser("/* a ** b */x")
	err := p.loudComment()
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 12 {
		t.Errorf("loudComment: pos = %d, want 12", p.scanner.Position())
	}
}

func TestLoudCommentUnterminated(t *testing.T) {
	p := newTestParser("/* unterminated")
	err := p.loudComment()
	if err == nil {
		t.Error("loudComment: expected error for unterminated comment")
	}
}

func TestLoudCommentNotComment(t *testing.T) {
	p := newTestParser("x")
	err := p.loudComment()
	if err == nil {
		t.Error("loudComment: expected error for non-comment")
	}
	if err.Error() != "expected \"/*\"." {
		t.Errorf("loudComment error = %q, want %q", err.Error(), "expected \"/*\".")
	}
}

// ==========================================================================
// Expect whitespace
// ==========================================================================

func TestExpectWhitespaceEof(t *testing.T) {
	p := newTestParser("")
	err := p.expectWhitespace(false)
	if err == nil {
		t.Error("expectWhitespace: expected error at EOF")
	}
	if err.Error() != "Expected whitespace." {
		t.Errorf("expectWhitespace error = %q, want %q", err.Error(), "Expected whitespace.")
	}
}

func TestExpectWhitespaceNoWs(t *testing.T) {
	p := newTestParser("abc")
	err := p.expectWhitespace(false)
	if err == nil {
		t.Error("expectWhitespace: expected error for non-whitespace, non-comment")
	}
	if err.Error() != "Expected whitespace." {
		t.Errorf("expectWhitespace error = %q, want %q", err.Error(), "Expected whitespace.")
	}
}

func TestExpectWhitespaceSpaces(t *testing.T) {
	p := newTestParser("  x")
	err := p.expectWhitespace(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 2 {
		t.Errorf("expectWhitespace: pos = %d, want 2", p.scanner.Position())
	}
}

func TestExpectWhitespaceComment(t *testing.T) {
	p := newTestParser("// cmt\nx")
	err := p.expectWhitespace(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 7 {
		t.Errorf("expectWhitespace: pos = %d, want 7", p.scanner.Position())
	}
}

func TestExpectWhitespaceMixed(t *testing.T) {
	p := newTestParser("  /* cmt */ x")
	err := p.expectWhitespace(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.scanner.Position() != 12 {
		t.Errorf("expectWhitespace: pos = %d, want 12", p.scanner.Position())
	}
}

// ==========================================================================
// Identifier edge cases
// ==========================================================================

func TestIdentifierEmpty(t *testing.T) {
	p := newTestParser("")
	_, err := p.identifier(false, false)
	if err == nil {
		t.Error("identifier: expected error for empty input")
	}
	if err.Error() != "Expected identifier." {
		t.Errorf("identifier error = %q, want %q", err.Error(), "Expected identifier.")
	}
}

func TestIdentifierDigitStart(t *testing.T) {
	p := newTestParser("1foo")
	_, err := p.identifier(false, false)
	if err == nil {
		t.Error("identifier: expected error for digit start")
	}
	if err.Error() != "Expected identifier." {
		t.Errorf("identifier error = %q, want %q", err.Error(), "Expected identifier.")
	}
}

func TestIdentifierUnitDashBeforeDot(t *testing.T) {
	// In unit mode, dash before '.' returns (stops at dash)
	p := newTestParser("foo-.")
	got, err := p.identifier(false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo" {
		t.Errorf("identifier (unit, dash before dot) = %q, want %q", got, "foo")
	}
}

func TestIdentifierEscapeOnly(t *testing.T) {
	p := newTestParser(`\41 `)
	got, err := p.identifier(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "A" {
		t.Errorf("identifier (escape-only) = %q, want %q", got, "A")
	}
}

func TestIdentifierUppercaseStart(t *testing.T) {
	p := newTestParser("Escaped")
	got, err := p.identifier(false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Escaped" {
		t.Errorf("identifier = %q, want %q", got, "Escaped")
	}
}

func TestIdentifierUnderscoreToDash(t *testing.T) {
	p := newTestParser("_var")
	got, err := p.identifier(true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "-var" {
		t.Errorf("identifier (normalized) = %q, want %q", got, "-var")
	}
}

func TestIdentifierBareDash(t *testing.T) {
	p := newTestParser("-")
	_, err := p.identifier(false, false)
	if err == nil {
		t.Error("identifier: expected error for bare dash")
	}
	if err.Error() != "Expected identifier." {
		t.Errorf("identifier error = %q, want %q", err.Error(), "Expected identifier.")
	}
}

func TestIdentifierBodyEscape(t *testing.T) {
	p := newTestParser("f\\41o")
	p.scanner.ScanChar('f')
	got, err := p.identifierBody()
	if err != nil {
		t.Fatal(err)
	}
	if got != "Ao" {
		t.Errorf("identifierBody (escape) = %q, want %q", got, "Ao")
	}
}

// ==========================================================================
// String edge cases
// ==========================================================================

func TestStringParserEmpty(t *testing.T) {
	p := newTestParser("")
	_, err := p.string()
	if err == nil {
		t.Error("string: expected error for empty input")
	}
}

func TestStringNotAQuote(t *testing.T) {
	p := newTestParser("abc")
	_, err := p.string()
	if err == nil {
		t.Error("string: expected error for non-quote start")
	}
	if err.Error() != "Expected string." {
		t.Errorf("string error = %q, want %q", err.Error(), "Expected string.")
	}
}

func TestStringUnterminatedSingle(t *testing.T) {
	p := newTestParser("'unterminated")
	_, err := p.string()
	if err == nil {
		t.Error("string: expected error for unterminated single-quoted string")
	}
	if err.Error() != "Expected '." {
		t.Errorf("string error = %q, want %q", err.Error(), "Expected '.")
	}
}

func TestStringUnterminatedDouble(t *testing.T) {
	p := newTestParser("\"unterminated")
	_, err := p.string()
	if err == nil {
		t.Error("string: expected error for unterminated double-quoted string")
	}
	if err.Error() != "Expected \"." {
		t.Errorf("string error = %q, want %q", err.Error(), "Expected \".")
	}
}

// ==========================================================================
// Natural number edge cases
// ==========================================================================

func TestNaturalNumberStopsAtNonDigit(t *testing.T) {
	p := newTestParser("1a")
	got, err := p.naturalNumber()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1.0 {
		t.Errorf("naturalNumber = %v, want 1.0", got)
	}
	if p.scanner.Position() != 1 {
		t.Errorf("naturalNumber: pos = %d, want 1", p.scanner.Position())
	}
}

// ==========================================================================
// Declaration value edge cases
// ==========================================================================

func TestDeclarationValueAllowEmptyTrue(t *testing.T) {
	p := newTestParser(";")
	got, err := p.declarationValue(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("declarationValue = %q, want %q", got, "")
	}
}

func TestDeclarationValueAllowEmptyFalse(t *testing.T) {
	p := newTestParser(";")
	_, err := p.declarationValue(false)
	if err == nil {
		t.Error("declarationValue: expected error for empty when allowEmpty=false")
	}
	if err.Error() != "Expected token." {
		t.Errorf("declarationValue error = %q, want %q", err.Error(), "Expected token.")
	}
}

func TestDeclarationValueEscape(t *testing.T) {
	p := newTestParser(`\41  b;`)
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "A b" {
		t.Errorf("declarationValue (escape) = %q, want %q", got, "A b")
	}
}

func TestDeclarationValueLoudComment(t *testing.T) {
	p := newTestParser("a /* cmt */ b;")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a /* cmt */ b" {
		t.Errorf("declarationValue (loud comment) = %q, want %q", got, "a /* cmt */ b")
	}
}

func TestDeclarationValueNewlineCollapse(t *testing.T) {
	p := newTestParser("a\n\nb;")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb" {
		t.Errorf("declarationValue (newline collapse) = %q, want %q", got, "a\nb")
	}
}

func TestDeclarationValueCrlf(t *testing.T) {
	p := newTestParser("a\r\nb;")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb" {
		t.Errorf("declarationValue (CRLF) = %q, want %q", got, "a\nb")
	}
}

func TestDeclarationValueQuadrupleNested(t *testing.T) {
	p := newTestParser("a(b(c(d)e)f);")
	got, err := p.declarationValue(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a(b(c(d)e)f)" {
		t.Errorf("declarationValue (quad nested) = %q, want %q", got, "a(b(c(d)e)f)")
	}
}

func TestDeclarationValueUnmatchedOpenBracket(t *testing.T) {
	p := newTestParser("a(b;")
	_, err := p.declarationValue(false)
	if err == nil {
		t.Error("declarationValue: expected error for unmatched opening bracket")
	}
}

// ==========================================================================
// URL edge cases
// ==========================================================================

func TestTryUrlFull(t *testing.T) {
	p := newTestParser("url(http://example.com)")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(http://example.com)" {
		t.Errorf("tryUrl = %q, want %q", got, "url(http://example.com)")
	}
}

func TestTryUrlEscape(t *testing.T) {
	p := newTestParser(`url(\41 bc)`)
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(Abc)" {
		t.Errorf("tryUrl (escape) = %q, want %q", got, "url(Abc)")
	}
}

func TestTryUrlNonAscii(t *testing.T) {
	p := newTestParser("url(\u00E9)")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(\u00E9)" {
		t.Errorf("tryUrl (non-ASCII) = %q, want %q", got, "url(\u00E9)")
	}
}

func TestTryUrlWhitespace(t *testing.T) {
	p := newTestParser("url(  foo  )bar")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(foo)" {
		t.Errorf("tryUrl (whitespace) = %q, want %q", got, "url(foo)")
	}
}

func TestTryUrlRestoreNoParen(t *testing.T) {
	p := newTestParser("urlNot")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("tryUrl (no paren) = %q, want empty", got)
	}
	if p.scanner.Position() != 0 {
		t.Errorf("tryUrl pos = %d, want 0", p.scanner.Position())
	}
}

func TestTryUrlRestoreInvalidChar(t *testing.T) {
	p := newTestParser("url(abc!)")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("tryUrl (invalid char) = %q, want empty", got)
	}
	if p.scanner.Position() != 0 {
		t.Errorf("tryUrl pos = %d, want 0", p.scanner.Position())
	}
}

func TestTryUrlCaseInsensitiveUrl(t *testing.T) {
	p := newTestParser("URL(foo)")
	got, err := p.tryUrl()
	if err != nil {
		t.Fatal(err)
	}
	if got != "url(foo)" {
		t.Errorf("tryUrl (URL) = %q, want %q", got, "url(foo)")
	}
}

// ==========================================================================
// Escape edge cases
// ==========================================================================

func TestEscapeSurrogate(t *testing.T) {
	p := newTestParser(`\D800 `)
	_, err := p.escape(false)
	if err == nil {
		t.Error("escape: expected error for surrogate code point")
	}
	if err.Error() != "Invalid Unicode code point." {
		t.Errorf("escape error = %q, want %q", err.Error(), "Invalid Unicode code point.")
	}
}

func TestEscapeDigitAtStart(t *testing.T) {
	p := newTestParser(`\31 `)
	got, err := p.escape(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != `\31 ` {
		t.Errorf("escape (digit at start) = %q, want %q", got, `\31 `)
	}
}

func TestEscapeControlChar7F(t *testing.T) {
	p := newTestParser(`\7F`)
	got, err := p.escape(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != `\7f ` {
		t.Errorf("escape (DEL) = %q, want %q", got, `\7f `)
	}
}

func TestEscapePlainSlash(t *testing.T) {
	p := newTestParser(`\g`)
	got, err := p.escape(true)
	if err != nil {
		t.Fatal(err)
	}
	// \g — 'g' is name-start, so the backslash is stripped and just 'g' returned
	if got != `g` {
		t.Errorf("escape (non-hex, name) = %q, want %q", got, `g`)
	}
}

func TestEscapeSixHexDigits(t *testing.T) {
	p := newTestParser(`\000041`)
	got, err := p.escape(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "A" {
		t.Errorf("escape (6 hex digits) = %q, want %q", got, "A")
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
	secondary := map[sasscommon.FileSpan]string{span: "secondary desc"}
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
	_, err := consumeEscapedCharacter(scanner)
	if err == nil {
		t.Error("consumeEscapedCharacter: expected error for EOF after \\")
	}
	if err.Error() != "Expected escape sequence." {
		t.Errorf("error = %q, want %q", err.Error(), "Expected escape sequence.")
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
