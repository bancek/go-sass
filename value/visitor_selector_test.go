package value

import (
	"testing"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/sourcemapbuffer"
)

// serializeAttributeSelectorForTest runs visitAttributeSelector in expanded
// mode and returns the CSS it writes.
func serializeAttributeSelectorForTest(t *testing.T, attr *AttributeSelector) string {
	t.Helper()
	sv := &SerializeVisitor{
		sb:              sourcemapbuffer.NewNoSourceMapBuffer(),
		quote:           true,
		lineFeed:        LineFeedLf,
		indentCharacter: ' ',
		indentWidth:     2,
	}
	if _, err := sv.VisitAttributeSelector(attr); err != nil {
		t.Fatal(err)
	}
	return sv.sb.String()
}

// An attribute selector whose value is the empty string is quoted rather than
// failing out of toCssIdentifier.
//
// Regression test for sass/dart-sass#2889.
// Matches Dart: _SerializeVisitor.visitAttributeSelector
func TestVisitAttributeSelectorEmptyValue(t *testing.T) {
	attr := NewAttributeSelectorWithOperator(
		NewQualifiedName("a"),
		AttributeOperatorEqual,
		"",
		sasscommon.BogusSpan,
		nil,
	)

	if css := serializeAttributeSelectorForTest(t, attr); css != `[a=""]` {
		t.Errorf("attribute selector = %q, want %q", css, `[a=""]`)
	}
}

// An identifier-valued attribute selector still prints unquoted.
func TestVisitAttributeSelectorIdentifierValue(t *testing.T) {
	attr := NewAttributeSelectorWithOperator(
		NewQualifiedName("a"),
		AttributeOperatorEqual,
		"foo",
		sasscommon.BogusSpan,
		nil,
	)

	if css := serializeAttributeSelectorForTest(t, attr); css != `[a=foo]` {
		t.Errorf("attribute selector = %q, want %q", css, `[a=foo]`)
	}
}
