package util

import "testing"

func TestIsDigit(t *testing.T) {
	for _, ch := range []int{'0', '1', '5', '9'} {
		if !IsDigit(ch) {
			t.Errorf("IsDigit(%q) = false", rune(ch))
		}
	}
	for _, ch := range []int{'a', 'z', 'A', 'Z', '_', '-'} {
		if IsDigit(ch) {
			t.Errorf("IsDigit(%q) = true", rune(ch))
		}
	}
}

func TestIsHex(t *testing.T) {
	for _, ch := range []int{'0', '9', 'a', 'f', 'A', 'F'} {
		if !IsHex(ch) {
			t.Errorf("IsHex(%q) = false", rune(ch))
		}
	}
	if IsHex('g') {
		t.Error("IsHex(g) = true")
	}
	if IsHex('G') {
		t.Error("IsHex(G) = true")
	}
}

func TestIsAlphabetic(t *testing.T) {
	for _, ch := range []int{'a', 'm', 'z', 'A', 'M', 'Z'} {
		if !IsAlphabetic(ch) {
			t.Errorf("IsAlphabetic(%q) = false", rune(ch))
		}
	}
	for _, ch := range []int{'0', '9', '_', '-'} {
		if IsAlphabetic(ch) {
			t.Errorf("IsAlphabetic(%q) = true", rune(ch))
		}
	}
}

func TestIsAlphanumeric(t *testing.T) {
	for _, ch := range []int{'a', 'z', 'A', 'Z', '0', '9'} {
		if !IsAlphanumeric(ch) {
			t.Errorf("IsAlphanumeric(%q) = false", rune(ch))
		}
	}
	if IsAlphanumeric('-') {
		t.Error("IsAlphanumeric(-) = true")
	}
	if IsAlphanumeric('_') {
		t.Error("IsAlphanumeric(_) = true")
	}
}

func TestIsNameStart(t *testing.T) {
	if !IsNameStart('_') {
		t.Error("IsNameStart(_) = false")
	}
	if !IsNameStart('a') {
		t.Error("IsNameStart(a) = false")
	}
	if !IsNameStart(0x0080) {
		t.Error("IsNameStart(U+0080) = false")
	}
	if IsNameStart('0') {
		t.Error("IsNameStart(0) = true")
	}
	if IsNameStart('-') {
		t.Error("IsNameStart(-) = true")
	}
}

func TestIsName(t *testing.T) {
	if !IsName('_') {
		t.Error("IsName(_) = false")
	}
	if !IsName('a') {
		t.Error("IsName(a) = false")
	}
	if !IsName('0') {
		t.Error("IsName(0) = false")
	}
	if !IsName('-') {
		t.Error("IsName(-) = false")
	}
	if !IsName(0x0080) {
		t.Error("IsName(U+0080) = false")
	}
	if IsName(' ') {
		t.Error("IsName( ) = true")
	}
	if IsName('.') {
		t.Error("IsName(.) = true")
	}
}

func TestIsWhitespace(t *testing.T) {
	for _, ch := range []int{' ', '\t', '\n', '\r', '\f'} {
		if !IsWhitespace(ch) {
			t.Errorf("IsWhitespace(%q) = false", rune(ch))
		}
	}
	if IsWhitespace('a') {
		t.Error("IsWhitespace(a) = true")
	}
}

func TestIsNewline(t *testing.T) {
	for _, ch := range []int{'\n', '\r', '\f'} {
		if !IsNewline(ch) {
			t.Errorf("IsNewline(%q) = false", rune(ch))
		}
	}
	if IsNewline(' ') {
		t.Error("IsNewline( ) = true")
	}
}

func TestIsSpaceOrTab(t *testing.T) {
	if !IsSpaceOrTab(' ') {
		t.Error("IsSpaceOrTab( ) = false")
	}
	if !IsSpaceOrTab('\t') {
		t.Error("IsSpaceOrTab(tab) = false")
	}
	if IsSpaceOrTab('\n') {
		t.Error("IsSpaceOrTab(newline) = true")
	}
}

func TestAsHex(t *testing.T) {
	tests := []struct {
		ch   int
		want int
	}{
		{'0', 0},
		{'9', 9},
		{'A', 10},
		{'F', 15},
		{'a', 10},
		{'f', 15},
	}
	for _, tt := range tests {
		got := AsHex(tt.ch)
		if got != tt.want {
			t.Errorf("AsHex(%q) = %d, want %d", rune(tt.ch), got, tt.want)
		}
	}
}

func TestHexCharFor(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, '0'},
		{9, '9'},
		{10, 'a'},
		{15, 'f'},
	}
	for _, tt := range tests {
		got := HexCharFor(tt.n)
		if got != tt.want {
			t.Errorf("HexCharFor(%d) = %q, want %q", tt.n, rune(got), rune(tt.want))
		}
	}
}

func TestHexCharForPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("HexCharFor(16) should panic")
		}
	}()
	HexCharFor(16)
}

func TestOpposite(t *testing.T) {
	tests := []struct {
		ch   int
		want int
	}{
		{'(', ')'},
		{'{', '}'},
		{'[', ']'},
	}
	for _, tt := range tests {
		got, err := Opposite(tt.ch)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("Opposite(%q) = %q, want %q", rune(tt.ch), rune(got), rune(tt.want))
		}
	}
}

func TestOppositeError(t *testing.T) {
	_, err := Opposite('x')
	if err == nil {
		t.Error("Opposite(x) should return error")
	}
}

func TestToUpperCase(t *testing.T) {
	tests := []struct {
		ch   int
		want int
	}{
		{'a', 'A'},
		{'z', 'Z'},
		{'A', 'A'},
		{'Z', 'Z'},
		{'0', '0'},
	}
	for _, tt := range tests {
		got := ToUpperCase(tt.ch)
		if got != tt.want {
			t.Errorf("ToUpperCase(%q) = %q, want %q", rune(tt.ch), rune(got), rune(tt.want))
		}
	}
}

func TestToLowerCase(t *testing.T) {
	tests := []struct {
		ch   int
		want int
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'a', 'a'},
		{'z', 'z'},
		{'0', '0'},
	}
	for _, tt := range tests {
		got := ToLowerCase(tt.ch)
		if got != tt.want {
			t.Errorf("ToLowerCase(%q) = %q, want %q", rune(tt.ch), rune(got), rune(tt.want))
		}
	}
}

func TestCharacterEqualsIgnoreCase(t *testing.T) {
	if !CharacterEqualsIgnoreCase('a', 'A') {
		t.Error("a and A should be equal ignore case")
	}
	if !CharacterEqualsIgnoreCase('Z', 'z') {
		t.Error("Z and z should be equal ignore case")
	}
	if CharacterEqualsIgnoreCase('a', 'b') {
		t.Error("a and b should not be equal")
	}
	if CharacterEqualsIgnoreCase('a', 'a') {
		t.Log("same case should be equal")
	}
	if CharacterEqualsIgnoreCase('0', '0') {
		t.Log("same digit should be equal")
	}
}

func TestIsPrivate(t *testing.T) {
	if !IsPrivate("-foo") {
		t.Error("-foo should be private")
	}
	if !IsPrivate("_foo") {
		t.Error("_foo should be private")
	}
	if IsPrivate("foo") {
		t.Error("foo should not be private")
	}
}
