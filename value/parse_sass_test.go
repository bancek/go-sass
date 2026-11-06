package value

import "testing"

func TestSassParserTree(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "00-empty",
			input: "",
			want:  "",
		},
		{
			name: "01-variable",
			input: `$color: red
`,
			want: "$color: red;",
		},
		{
			name: "02-rule",
			input: `a
  color: red
`,
			want: "a {color: red;}",
		},
		{
			name: "03-nesting",
			input: `nav
  > ul
    > li
      color: red
`,
			want: "nav {> ul {> li {color: red;}}}",
		},
		{
			name: "04-comments",
			input: `// This is a silent comment
a
  color: red

// Another comment

/* A loud comment */
b
  color: blue
`,
			want: `// This is a silent comment
 a {color: red;} // Another comment
 /* A loud comment */ b {color: blue;}`,
		},
		{
			name: "05-mixin",
			input: `=my-mixin
  color: red

a
  +my-mixin
`,
			want: "@mixin my-mixin {color: red;} a {@include my-mixin;}",
		},
		{
			name: "06-for-loop",
			input: `@for $i from 1 through 3
  .item-#{$i}
    width: 10px * $i
`,
			want: "@for $i from 1 through 3 {.item-#{$i} {width: 10px * $i;}}",
		},
		{
			name: "07-interpolation",
			input: `$name: sidebar
.#{$name}
  width: 200px
`,
			want: "$name: sidebar; .#{$name} {width: 200px;}",
		},
		{
			name: "08-media",
			input: `.sidebar
  @media screen and (min-width: 768px)
    width: 200px
`,
			want: ".sidebar {@media screen  and (#{min-width}: #{768px}) {width: 200px;}}",
		},
		{
			name: "09-function",
			input: `@function double($n)
  @return $n * 2

a
  width: double(10px)
`,
			want: "@function double($n) {@return $n * 2;} a {width: double(10px);}",
		},
		{
			name: "10-each-while",
			input: `@each $i in 1, 2, 3
  .item-#{$i}
    width: 10px * $i

$i: 1
@while $i < 4
  .while-#{$i}
    width: 10px * $i
  $i: $i + 1
`,
			want: "@each $i in 1, 2, 3 {.item-#{$i} {width: 10px * $i;}} $i: 1; @while $i < 4 {.while-#{$i} {width: 10px * $i;} $i: $i + 1;}",
		},
		{
			name: "basic",
			input: `// Variables
$color: red
$size: 10px

// Mixin
=my-mixin
  color: $color

// Style rules
a
  color: $color
  font-size: $size

  +my-mixin

  // Nested
  > span
    font-weight: bold

    @media screen
      text-decoration: underline

  // Interpolation
  &--#{$color}
    background: blue

// Function
@function double($n)
  @return $n * 2

b
  width: double(20px)

// For loop
@for $i from 1 through 3
  .for-#{$i}
    width: 10px * $i
`,
			want: `// Variables
 $color: red; $size: 10px; // Mixin
 @mixin my-mixin {color: $color;} // Style rules
 a {color: $color; font-size: $size; @include my-mixin; // Nested
 > span {font-weight: bold; @media screen {text-decoration: underline;}} // Interpolation
 &--#{$color} {background: blue;}} // Function
 @function double($n) {@return $n * 2;} b {width: double(20px);} // For loop
 @for $i from 1 through 3 {.for-#{$i} {width: 10px * $i;}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stylesheet, err := NewSassParser([]byte(tt.input), nil, false).Parse()
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
