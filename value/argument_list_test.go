package value

import (
	"testing"

	"github.com/bancek/go-sass/orderedmap"
)

func TestArgListConstruction(t *testing.T) {
	kw := orderedmap.New[string, Value]()
	kw.Put("a", NewUnitlessNumber(1))
	al, err := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, kw, ListSeparatorSpace)
	if err != nil {
		t.Fatal(err)
	}
	if al.LengthAsList() != 2 {
		t.Errorf("LengthAsList = %d, want 2", al.LengthAsList())
	}
	if al.Separator() != ListSeparatorSpace {
		t.Error("separator should be Space")
	}
}

func TestArgListKeywords(t *testing.T) {
	kw := orderedmap.New[string, Value]()
	kw.Put("a", NewUnitlessNumber(1))
	al, _ := NewSassArgumentList([]Value{}, kw, ListSeparatorSpace)
	if al.WereKeywordsAccessed() {
		t.Error("wereKeywordsAccessed should start false")
	}
	kws := al.Keywords()
	if kws == nil {
		t.Error("Keywords should not be nil")
	}
	if !al.WereKeywordsAccessed() {
		t.Error("wereKeywordsAccessed should be true after Keywords()")
	}
}

func TestArgListKeywordsWithoutMarking(t *testing.T) {
	kw := orderedmap.New[string, Value]()
	kw.Put("a", NewUnitlessNumber(1))
	al, _ := NewSassArgumentList([]Value{}, kw, ListSeparatorSpace)
	_ = al.KeywordsWithoutMarking()
	if al.WereKeywordsAccessed() {
		t.Error("KeywordsWithoutMarking should not set wereKeywordsAccessed")
	}
}

func TestArgListDelegatesToList(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorComma)
	if al.Separator() != ListSeparatorComma {
		t.Error("separator should delegate to list")
	}
	if al.HasBrackets() {
		t.Error("argument list should not have brackets")
	}
	list, err := al.AsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("AsList length = %d, want 2", len(list))
	}
	if al.LengthAsList() != 2 {
		t.Error("LengthAsList should be 2")
	}
}

func TestArgListEquals(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorSpace)
	l, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	if !al.Equals(l) {
		t.Error("argument list should equal list with same contents")
	}
	if !l.Equals(al) {
		t.Error("list should equal argument list with same contents")
	}
	al2, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1)}, nil, ListSeparatorSpace)
	if al.Equals(al2) {
		t.Error("different contents should not be equal")
	}
}

func TestArgListHashCode(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorSpace)
	l, _ := NewSassList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, ListSeparatorSpace, false)
	if al.HashCode() != l.HashCode() {
		t.Error("argument list hash should match list hash")
	}
}

func TestArgListToCssString(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorSpace)
	got, err := al.ToCssString(true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "1 2" {
		t.Errorf("ToCssString = %q, want %q", got, "1 2")
	}
}

func TestArgListString(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1), NewUnitlessNumber(2)}, nil, ListSeparatorSpace)
	got, err := al.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != "(1 2)" {
		t.Errorf("String = %q, want %q", got, "(1 2)")
	}
}

func TestArgListOperators(t *testing.T) {
	al, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1)}, nil, ListSeparatorSpace)
	_, err := al.Plus(NewUnitlessNumber(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = al.Times(NewUnitlessNumber(1))
	if err == nil {
		t.Error("Times should error")
	}
}

func TestArgListTryMap(t *testing.T) {
	empty, _ := NewSassArgumentList(nil, nil, ListSeparatorComma)
	if empty.TryMap() == nil {
		t.Error("empty argument list TryMap() should return an empty map")
	}
	if empty.TryMap().LengthAsList() != 0 {
		t.Error("empty argument list TryMap() should be empty")
	}
	nonEmpty, _ := NewSassArgumentList([]Value{NewUnitlessNumber(1)}, nil, ListSeparatorComma)
	if nonEmpty.TryMap() != nil {
		t.Error("non-empty argument list TryMap() should return nil")
	}
}
