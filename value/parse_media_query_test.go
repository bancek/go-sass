package value

import (
	"strings"
	"testing"
)

func newTestCssMediaQueryParser(text string) *CssMediaQueryParser {
	return &CssMediaQueryParser{Parser: *NewParser([]byte(text), nil, nil)}
}

func TestMediaQueryTypeOnly(t *testing.T) {
	p := newTestCssMediaQueryParser("screen")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}
	q := queries[0]
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("Type = %v, want 'screen'", q.Type)
	}
}

func TestMediaQueryTypeWithModifier(t *testing.T) {
	p := newTestCssMediaQueryParser("only screen")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if q.Modifier == nil || *q.Modifier != "only" {
		t.Errorf("Modifier = %v, want 'only'", q.Modifier)
	}
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("Type = %v, want 'screen'", q.Type)
	}
}

func TestMediaQueryNotCondition(t *testing.T) {
	p := newTestCssMediaQueryParser("not (color)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if len(q.Conditions) != 1 || q.Conditions[0] != "(not (color))" {
		t.Errorf("Conditions = %v, want [\"(not (color))\"]", q.Conditions)
	}
}

func TestMediaQueryCondition(t *testing.T) {
	p := newTestCssMediaQueryParser("(color)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if len(q.Conditions) != 1 || q.Conditions[0] != "(color)" {
		t.Errorf("Conditions = %v, want [\"(color)\"]", q.Conditions)
	}
}

func TestMediaQueryAnd(t *testing.T) {
	p := newTestCssMediaQueryParser("screen and (color)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("Type = %v, want 'screen'", q.Type)
	}
	if len(q.Conditions) != 1 || q.Conditions[0] != "(color)" {
		t.Errorf("Conditions = %v, want [\"(color)\"]", q.Conditions)
	}
}

func TestMediaQueryOr(t *testing.T) {
	p := newTestCssMediaQueryParser("(color) or (monochrome)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if q.Conjunction {
		t.Error("expected conjunction=false for 'or'")
	}
	if len(q.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(q.Conditions))
	}
}

func TestMediaQueryCommaSeparated(t *testing.T) {
	p := newTestCssMediaQueryParser("screen, print")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}
}

func TestMediaQueryModifierAndNot(t *testing.T) {
	p := newTestCssMediaQueryParser("screen and not (color)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if q.Modifier != nil {
		t.Errorf("Modifier = %v, want nil", q.Modifier)
	}
	if len(q.Conditions) != 1 || q.Conditions[0] != "(not (color))" {
		t.Errorf("Conditions = %v, want [\"(not (color))\"]", q.Conditions)
	}
}

func TestMediaQueryOnlyAndNot(t *testing.T) {
	p := newTestCssMediaQueryParser("only screen and not (color)")
	queries, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	if q.Modifier == nil || *q.Modifier != "only" {
		t.Errorf("Modifier = %v, want 'only'", q.Modifier)
	}
	if q.Type == nil || *q.Type != "screen" {
		t.Errorf("Type = %v, want 'screen'", q.Type)
	}
	if len(q.Conditions) != 1 || q.Conditions[0] != "(not (color))" {
		t.Errorf("Conditions = %v, want [\"(not (color))\"]", q.Conditions)
	}
}

func TestMediaQuerySpan(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"screen", "screen"},
		{"screen ", "screen"},
		{"only screen ", "only screen"},
		{"(color) ", "(color)"},
		{"screen and (color) ", "screen and (color)"},
		{"screen and not (color)", "screen and not (color)"},
		{"not (color)", "not (color)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := newTestCssMediaQueryParser(tt.input)
			queries, err := p.Parse()
			if err != nil {
				t.Fatal(err)
			}
			if len(queries) != 1 {
				t.Fatalf("expected 1 query, got %d", len(queries))
			}
			span, err := queries[0].Span()
			if err != nil {
				t.Fatal(err)
			}
			got, err := span.SpanText()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("span = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMediaQueryErrorEmpty(t *testing.T) {
	p := newTestCssMediaQueryParser("")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for empty input")
	}
	want := strings.Join([]string{`Error: Expected identifier.`, `  ╷`, `1 │ `, `  │ ^`, `  ╵`, `  - 1:1  root stylesheet`}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
