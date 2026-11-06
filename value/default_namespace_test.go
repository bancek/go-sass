package value

import (
	"testing"

	"github.com/bancek/go-sass/sassurl"
)

func TestDefaultNamespace(t *testing.T) {
	tests := []struct {
		url   string
		want  string
		valid bool
	}{
		{"file:///home/user/_bootstrap.scss", "bootstrap", true},
		{"file:///home/user/my_utils.scss", "my_utils", true},
		{"file:///home/user/theme.css.scss", "theme", true},
		{"file:///home/user/1_bad.scss", "1_bad", false},
		{"file:///home/user/dep%20file.scss", "dep file", false},
	}
	for _, tt := range tests {
		u, err := sassurl.Parse(tt.url)
		if err != nil {
			t.Fatal(err)
		}
		got, valid := DefaultNamespace(u)
		if got != tt.want || valid != tt.valid {
			t.Errorf("DefaultNamespace(%q) = %q, %v; want %q, %v", tt.url, got, valid, tt.want, tt.valid)
		}
	}
}
