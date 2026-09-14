package value

import (
	"testing"
)

func TestCssParserUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "identifier with accent",
			input: `.café { color: red }`,
			want:  `.café  {color: red;}`,
		},
		{
			name:  "identifier with umlaut",
			input: `.München { color: red }`,
			want:  `.München  {color: red;}`,
		},
		{
			name:  "attribute selector unicode",
			input: `div[title="čas"] { color: red }`,
			want:  `div[title="čas"]  {color: red;}`,
		},
		{
			name:  "attribute selector japanese",
			input: `[data-名前="値"] { color: red }`,
			want:  `[data-名前="値"]  {color: red;}`,
		},
		{
			name:  "quoted string japanese",
			input: `.a { content: "日本語" }`,
			want:  `.a  {content: "日本語";}`,
		},
		{
			name:  "quoted string with eszett",
			input: `.a { content: "größe" }`,
			want:  `.a  {content: "größe";}`,
		},
		{
			name:  "custom property naive",
			input: `.a { --naïve: "größe" }`,
			want:  `.a  {--naïve: "größe" ;}`,
		},
		{
			name:  "emoji in string",
			input: `.a { content: "hello 👋" }`,
			want:  `.a  {content: "hello 👋";}`,
		},
		{
			name:  "url with unicode",
			input: `.a { background: url(path/café.png) }`,
			want:  `.a  {background: url(path/café.png);}`,
		},
		{
			name: "combined unicode",
			input: `.café[title="čas"] {
  --naïve: "größe"
}`,
			want: ".café[title=\"čas\"]  {--naïve: \"größe\"\n;}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stylesheet, err := NewCssParser([]byte(tt.input), nil, false, nil).Parse()
			if err != nil {
				t.Fatal(err)
			}
			got, err := stylesheet.String()
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("mismatch\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestScssParserUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "scss variable with korean",
			input: `$변수: 1; a { b: $변수; }`,
		},
		{
			name:  "scss identifier with accent",
			input: `.café { color: red; }`,
		},
		{
			name: "scss combined",
			input: `.café {
  --naïve: "größe";
  content: "日本語";
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewScssParser([]byte(tt.input), nil, false).Parse()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSassParserUnicode(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "sass variable with korean",
			input: "$변수: 1\na\n\tb: $변수",
		},
		{
			name: "sass combined",
			input: `.café
	--naïve: "größe"
	content: "日本語"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSassParser([]byte(tt.input), nil, false).Parse()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
