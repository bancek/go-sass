package value

import "testing"

func TestCombinatorString(t *testing.T) {
	tests := []struct {
		c    Combinator
		want string
	}{
		{CombinatorNextSibling, "+"},
		{CombinatorChild, ">"},
		{CombinatorFollowingSibling, "~"},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.want {
			t.Errorf("Combinator(%d).String() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestCombinatorValuesDistinct(t *testing.T) {
	if CombinatorNextSibling == CombinatorChild {
		t.Error("NextSibling and Child should be distinct")
	}
	if CombinatorNextSibling == CombinatorFollowingSibling {
		t.Error("NextSibling and FollowingSibling should be distinct")
	}
	if CombinatorChild == CombinatorFollowingSibling {
		t.Error("Child and FollowingSibling should be distinct")
	}
}

func TestCombinatorDefaultString(t *testing.T) {
	c := Combinator(99)
	if got := c.String(); got != "?" {
		t.Errorf("unknown Combinator.String() = %q, want ?", got)
	}
}
