package value

import (
	"testing"

	"github.com/bancek/go-sass/sourcemapbuffer"
)

func serializeColorStyle(t *testing.T, c *SassColor, style OutputStyle) string {
	t.Helper()
	sv := &SerializeVisitor{
		sb:              sourcemapbuffer.NewNoSourceMapBuffer(),
		style:           style,
		quote:           true,
		lineFeed:        LineFeedLf,
		indentCharacter: ' ',
		indentWidth:     2,
	}
	if _, err := c.AcceptVoid(sv); err != nil {
		t.Fatal(err)
	}
	return sv.sb.String()
}

// Legacy rgb() with a non-integer channel uses percentages for all channels
// (#2800) — older browsers only accept integers or percentages here.
func TestFractionalChannelsUsePercents(t *testing.T) {
	c, err := NewColorRGB(0.0, 255.0, 127.5, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if got := serializeColorStyle(t, c, OutputStyleExpanded); got != "rgb(0%, 100%, 50%)" {
		t.Errorf("got %q, want %q", got, "rgb(0%, 100%, 50%)")
	}
}

func TestFractionalChannelsUsePercentsRgba(t *testing.T) {
	c, err := NewColorRGB(0.0, 255.0, 127.5, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if got := serializeColorStyle(t, c, OutputStyleExpanded); got != "rgba(0%, 100%, 50%, 0.5)" {
		t.Errorf("got %q, want %q", got, "rgba(0%, 100%, 50%, 0.5)")
	}
}

func TestIntegerChannelsKeepPlainSpelling(t *testing.T) {
	// Non-opaque so serialization goes through writeRgb rather than the
	// opaque hex/named-colour shortcut.
	c, err := NewColorRGB(0.0, 255.0, 128.0, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if got := serializeColorStyle(t, c, OutputStyleExpanded); got != "rgba(0, 255, 128, 0.5)" {
		t.Errorf("got %q, want %q", got, "rgba(0, 255, 128, 0.5)")
	}
}

// Non-integer legacy channels serialize as percentages (#2800), so the
// compressed rgb-vs-hsl tiebreak compares full captured function strings
// (Dart `_capture`). rgb(0.1, 0.1, 0.1) can no longer win as rgb(.1,.1,.1);
// HSL wins instead.
func TestCompressedTiebreakUsesCapturedStrings(t *testing.T) {
	// rgb(0.1, 0.1, 0.1) in the RGB space (0-255 channels).
	c, err := NewColorRGB(0.1, 0.1, 0.1, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if got := serializeColorStyle(t, c, OutputStyleCompressed); got != "hsl(0,0%,.0392156863%)" {
		t.Errorf("got %q, want %q", got, "hsl(0,0%,.0392156863%)")
	}
}
