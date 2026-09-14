package value

import (
	"testing"
)

func newTestAtRootQueryParser(text string) *AtRootQueryParser {
	return &AtRootQueryParser{Parser: *NewParser([]byte(text), nil, nil)}
}

func TestAtRootQueryParseWith(t *testing.T) {
	p := newTestAtRootQueryParser("(with: rule)")
	q, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if !q.Include {
		t.Error("expected include=true")
	}
	// (with: rule) includes style rules, so excludesStyleRules() is false
	if q.ExcludesStyleRules() {
		t.Error("expected excludesStyleRules()=false for (with: rule) — style rules are included")
	}
}

func TestAtRootQueryParseWithout(t *testing.T) {
	p := newTestAtRootQueryParser("(without: media)")
	q, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if q.Include {
		t.Error("expected include=false")
	}
	if !q.ExcludesName("media") {
		t.Error("expected excludesName(\"media\")=true")
	}
	if q.ExcludesName("supports") {
		t.Error("expected excludesName(\"supports\")=false")
	}
}

func TestAtRootQueryParseMultiple(t *testing.T) {
	p := newTestAtRootQueryParser("(with: rule media)")
	q, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if !q.Include {
		t.Error("expected include=true")
	}
	// (with: rule media) includes rule+media, so these are NOT excluded
	if q.ExcludesStyleRules() {
		t.Error("expected excludesStyleRules()=false — rule is included")
	}
	if q.ExcludesName("media") {
		t.Error("expected excludesName(\"media\")=false — media is included")
	}
	// supports is not in the include set, so it IS excluded
	if !q.ExcludesName("supports") {
		t.Error("expected excludesName(\"supports\")=true — not in include set")
	}
}

func TestAtRootQueryParseCaseInsensitive(t *testing.T) {
	p := newTestAtRootQueryParser("(without: MEDIA)")
	q, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if !q.ExcludesName("media") {
		t.Error("expected excludesName(\"media\")=true for case-insensitive input")
	}
}

func TestAtRootQueryParseErrorNoParen(t *testing.T) {
	p := newTestAtRootQueryParser("with: rule")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for missing paren")
	}
}

func TestAtRootQueryParseErrorEmpty(t *testing.T) {
	p := newTestAtRootQueryParser("")
	_, err := p.Parse()
	if err == nil {
		t.Error("expected error for empty input")
	}
}
