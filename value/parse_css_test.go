package value

import "testing"

func TestCssParserGolden(t *testing.T) {
	for _, name := range cssFixtures(t) {
		t.Run(name, func(t *testing.T) {
			css, want := loadFixture(t, "css/"+name)
			stylesheet, err := NewCssParser(css, nil, false, nil).Parse()
			if err != nil {
				t.Fatal(err)
			}
			got, err := stylesheet.String()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("%s: mismatch\nwant: %q\ngot:  %q", name, want, got)
			}
		})
	}
}
