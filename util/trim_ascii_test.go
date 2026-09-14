package util

import "testing"

func TestTrimAscii(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		excludeEscape bool
		want          string
	}{
		{
			name:          "basic trim",
			input:         "  foo  ",
			excludeEscape: false,
			want:          "foo",
		},
		{
			name:          "tab and newline",
			input:         "\t\nfoo\r\n",
			excludeEscape: false,
			want:          "foo",
		},
		{
			name:          "all whitespace",
			input:         "   \t\n  ",
			excludeEscape: false,
			want:          "",
		},
		{
			name:          "exclude escape preserves trailing whitespace",
			input:         "  foo\\   ",
			excludeEscape: true,
			want:          "foo\\ ",
		},
		{
			name:          "no excludeEscape trims trailing whitespace",
			input:         "  foo\\   ",
			excludeEscape: false,
			want:          "foo\\",
		},
		{
			name:          "nothing to trim",
			input:         "foo",
			excludeEscape: false,
			want:          "foo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimAscii(tt.input, tt.excludeEscape); got != tt.want {
				t.Errorf("TrimAscii(%q, %v) = %q, want %q", tt.input, tt.excludeEscape, got, tt.want)
			}
		})
	}
}
