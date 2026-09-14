package value

import (
	"strings"
	"testing"
)

func TestNewSassList(t *testing.T) {
	_, err := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorUndecided, false)
	if err == nil {
		t.Error("multi-element list with Undecided separator should error")
	}
	l, err := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	if err != nil {
		t.Fatal(err)
	}
	if l.separator != ListSeparatorSpace {
		t.Error("separator should be stored")
	}

	l2, err := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorUndecided, false)
	if err != nil {
		t.Fatal(err)
	}
	if l2.LengthAsList() != 1 {
		t.Error("single-element list with Undecided should be OK")
	}
}

func TestNewSassListEmpty(t *testing.T) {
	l := NewSassListEmpty(nil, nil)
	if l.separator != ListSeparatorUndecided {
		t.Error("default separator should be Undecided")
	}
	if l.HasBrackets() {
		t.Error("default brackets should be false")
	}
	sep := ListSeparatorComma
	b := true
	l2 := NewSassListEmpty(&sep, &b)
	if l2.separator != ListSeparatorComma {
		t.Error("separator should be Comma")
	}
	if !l2.HasBrackets() {
		t.Error("brackets should be true")
	}
}

func TestListEquals(t *testing.T) {
	l1, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	l2, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	if !l1.Equals(l2) {
		t.Error("identical lists should be equal")
	}
	l3, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorComma, false)
	if l1.Equals(l3) {
		t.Error("lists with different separators should not be equal")
	}
	l4, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, true)
	if l1.Equals(l4) {
		t.Error("lists with different brackets should not be equal")
	}
	if l1.Equals(NewUnitlessNumber(1)) {
		t.Error("list should not equal number")
	}
}

func TestListEqualsEmptyVsMap(t *testing.T) {
	emptyList, _ := NewSassList(nil, ListSeparatorSpace, false)
	emptyMap := EmptySassMap()
	if !emptyList.Equals(emptyMap) {
		t.Error("empty list should equal empty map")
	}
	nonEmpty, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, false)
	if nonEmpty.Equals(emptyMap) {
		t.Error("non-empty list should not equal empty map")
	}
}

func TestListEqualsVsArgList(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorSpace)
	if !l.Equals(al) {
		t.Error("list should equal argument list with same contents")
	}
}

func TestListHashCode(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	h1 := l.HashCode()
	h2 := l.HashCode()
	if h1 != h2 {
		t.Error("HashCode should be cached and deterministic")
	}
	l2, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorComma, false)
	if h1 != l2.HashCode() {
		t.Error("HashCode should ignore separator (contents only)")
	}
}

func TestListIsBlank(t *testing.T) {
	bracketed, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, true)
	if bracketed.IsBlank() {
		t.Error("bracketed list should never be blank")
	}
	blank, _ := NewSassList([]Value{&SassString{Text: "", HasQuotes: false}, &SassString{Text: "", HasQuotes: false}}, ListSeparatorSpace, false)
	if !blank.IsBlank() {
		t.Error("list with all blank elements should be blank")
	}
	nonBlank, _ := NewSassList([]Value{&SassString{Text: "", HasQuotes: false}, NewUnitlessNumber(1)}, ListSeparatorSpace, false)
	if nonBlank.IsBlank() {
		t.Error("list with non-blank element should not be blank")
	}
}

func TestListAsList(t *testing.T) {
	contents := []Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}
	l, _ := NewSassList(contents, ListSeparatorSpace, false)
	result, err := l.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != len(contents) {
		t.Error("AsList should return contents")
	}
}

func TestListSeparatorEnum(t *testing.T) {
	if ListSeparatorSpace.String() != "space" {
		t.Errorf("Space.String() = %q", ListSeparatorSpace.String())
	}
	if ListSeparatorComma.String() != "comma" {
		t.Errorf("Comma.String() = %q", ListSeparatorComma.String())
	}
	if ListSeparatorSlash.String() != "slash" {
		t.Errorf("Slash.String() = %q", ListSeparatorSlash.String())
	}
	if ListSeparatorUndecided.String() != "undecided" {
		t.Errorf("Undecided.String() = %q", ListSeparatorUndecided.String())
	}

	sep := ListSeparatorSpace.Separator()
	if sep == nil || *sep != " " {
		t.Errorf("Space.Separator() = %v, want \" \"", sep)
	}
	sep = ListSeparatorComma.Separator()
	if sep == nil || *sep != "," {
		t.Errorf("Comma.Separator() = %v, want \",\"", sep)
	}
	sep = ListSeparatorSlash.Separator()
	if sep == nil || *sep != "/" {
		t.Errorf("Slash.Separator() = %v, want \"/\"", sep)
	}
	if ListSeparatorUndecided.Separator() != nil {
		t.Error("Undecided.Separator() should be nil")
	}
}

func TestListTryMap(t *testing.T) {
	empty, _ := NewSassList(nil, ListSeparatorSpace, false)
	m := empty.TryMap()
	if m == nil {
		t.Error("empty list.TryMap() should return empty map")
	}
	nonEmpty, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, false)
	if nonEmpty.TryMap() != nil {
		t.Error("non-empty list.TryMap() should return nil")
	}
}

func TestListToCssString(t *testing.T) {
	spaceList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)}, ListSeparatorSpace, false)
	got, err := spaceList.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1 2 3" {
		t.Errorf("space list CSS = %q, want %q", got, "1 2 3")
	}

	commaList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)}, ListSeparatorComma, false)
	got, err = commaList.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1, 2, 3" {
		t.Errorf("comma list CSS = %q, want %q", got, "1, 2, 3")
	}

	slashList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSlash, false)
	got, err = slashList.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1 / 2" {
		t.Errorf("slash list CSS = %q, want %q", got, "1 / 2")
	}

	bracketedList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, true)
	got, err = bracketedList.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[1 2]" {
		t.Errorf("bracketed list CSS = %q, want %q", got, "[1 2]")
	}
}

func TestListToCssString_SingleElement(t *testing.T) {
	single, _ := NewSassList([]Value{NewUnitlessNumber(42)}, ListSeparatorComma, false)
	got, err := single.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("single comma list CSS = %q, want %q", got, "42")
	}

	single2, _ := NewSassList([]Value{NewUnitlessNumber(42)}, ListSeparatorSpace, false)
	got, err = single2.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "42" {
		t.Errorf("single space list CSS = %q, want %q", got, "42")
	}
}

func TestListToCssString_EmptyError(t *testing.T) {
	empty, _ := NewSassList(nil, ListSeparatorSpace, false)
	_, err := empty.ToCssString(false)
	if err == nil {
		t.Error("empty unbracketed list ToCssString should error")
	}
	if !strings.Contains(err.Error(), "isn't a valid CSS value") {
		t.Errorf("error should mention invalid CSS value, got: %v", err)
	}
}

func TestListToCssString_EmptyBracketed(t *testing.T) {
	empty, _ := NewSassList(nil, ListSeparatorSpace, true)
	got, err := empty.ToCssString(false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "[]" {
		t.Errorf("empty bracketed list CSS = %q, want %q", got, "[]")
	}
}

func TestListString(t *testing.T) {
	commaList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)}, ListSeparatorComma, false)
	got, err := commaList.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(1, 2, 3)" {
		t.Errorf("comma list inspect = %q, want %q", got, "(1, 2, 3)")
	}

	spaceList, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2), NewUnitlessNumber(3)}, ListSeparatorSpace, false)
	got, err = spaceList.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(1 2 3)" {
		t.Errorf("space list inspect = %q, want %q", got, "(1 2 3)")
	}

	empty, _ := NewSassList(nil, ListSeparatorSpace, false)
	got, err = empty.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "()" {
		t.Errorf("empty list inspect = %q, want %q", got, "()")
	}
}

func TestListString_SingleCommaInspect(t *testing.T) {
	single, _ := NewSassList([]Value{NewUnitlessNumber(42)}, ListSeparatorComma, false)
	got, err := single.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(42,)" {
		t.Errorf("single comma inspect = %q, want %q", got, "(42,)")
	}
}

func TestListOperators(t *testing.T) {
	l, _ := NewSassList([]Value{NewUnitlessNumber(1)}, ListSeparatorSpace, false)
	_, err := l.Plus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = l.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error for lists")
	}
}
