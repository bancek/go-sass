package value

import (
	"strings"
	"testing"
)

func newTestKeyframeSelectorParser(text string) *KeyframeSelectorParser {
	return &KeyframeSelectorParser{Parser: *NewParser([]byte(text), nil, nil)}
}

func TestKeyframeParseFrom(t *testing.T) {
	p := newTestKeyframeSelectorParser("from")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(selectors) != 1 || selectors[0] != "from" {
		t.Errorf("Parse() = %v, want [\"from\"]", selectors)
	}
}

func TestKeyframeParseTo(t *testing.T) {
	p := newTestKeyframeSelectorParser("to")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(selectors) != 1 || selectors[0] != "to" {
		t.Errorf("Parse() = %v, want [\"to\"]", selectors)
	}
}

func TestKeyframeParsePercentage(t *testing.T) {
	p := newTestKeyframeSelectorParser("50%")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(selectors) != 1 || selectors[0] != "50%" {
		t.Errorf("Parse() = %v, want [\"50%%\"]", selectors)
	}
}

func TestKeyframeParsePercentageDecimal(t *testing.T) {
	p := newTestKeyframeSelectorParser("12.5%")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(selectors) != 1 || selectors[0] != "12.5%" {
		t.Errorf("Parse() = %v, want [\"12.5%%\"]", selectors)
	}
}

func TestKeyframeParsePercentageWithPlus(t *testing.T) {
	p := newTestKeyframeSelectorParser("+50%")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if selectors[0] != "+50%" {
		t.Errorf("Parse() = %v, want [\"+50%%\"]", selectors)
	}
}

func TestKeyframeParsePercentageExponent(t *testing.T) {
	p := newTestKeyframeSelectorParser("1e2%")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if selectors[0] != "1e2%" {
		t.Errorf("Parse() = %v, want [\"1e2%%\"]", selectors)
	}
}

func TestKeyframeParsePercentageExponentCapital(t *testing.T) {
	p := newTestKeyframeSelectorParser("3E4%")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if selectors[0] != "3e4%" {
		t.Errorf("Parse() = %v, want [\"3e4%%\"]", selectors)
	}
}

func TestKeyframeParseCommaSeparated(t *testing.T) {
	p := newTestKeyframeSelectorParser("from, to")
	selectors, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(selectors) != 2 || selectors[0] != "from" || selectors[1] != "to" {
		t.Errorf("Parse() = %v, want [\"from\", \"to\"]", selectors)
	}
}

func TestKeyframeParseError(t *testing.T) {
	p := newTestKeyframeSelectorParser("foo")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for non-keyword, non-percentage input")
	}
	want := strings.Join([]string{`Error: Expected "to" or "from".`, `  ╷`, `1 │ foo`, `  │ ^`, `  ╵`, `  - 1:1  root stylesheet`}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestKeyframeParseErrorEmpty(t *testing.T) {
	p := newTestKeyframeSelectorParser("")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for empty input")
	}
	want := strings.Join([]string{`Error: Expected number.`, `  ╷`, `1 │ `, `  │ ^`, `  ╵`, `  - 1:1  root stylesheet`}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
