package value

import (
	"testing"
)

func TestSassParserGoldenFixtures(t *testing.T) {
	for _, name := range sassFixtures(t) {
		t.Run(name, func(t *testing.T) {
			sass, want := loadFixture(t, "sass/"+name)
			stylesheet, err := NewSassParser(sass, nil, false).Parse()
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
