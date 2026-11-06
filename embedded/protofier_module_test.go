package embedded

import (
	"testing"

	"github.com/bancek/go-sass/sassmodule"
	"github.com/bancek/go-sass/value"
)

func TestProtofierModuleRoundtrip(t *testing.T) {
	fns := NewOpaqueRegistry()
	mixins := NewOpaqueRegistry()
	modules := NewOpaqueRegistry()
	p := NewProtofier(nil, fns, mixins, modules)

	original := value.NewSassModule(sassmodule.NewBuiltInModule("meta", nil, nil, nil))
	protoValue, err := p.Protofy(original)
	if err != nil {
		t.Fatal(err)
	}
	if protoValue.GetCompilerModule() == nil {
		t.Fatalf("Protofy(module) = %T, want CompilerModule", protoValue.GetValue())
	}
	decoded, err := p.deprotofy(protoValue)
	if err != nil {
		t.Fatal(err)
	}
	module, ok := decoded.(*value.SassModule)
	if !ok {
		t.Fatalf("deprotofy(CompilerModule) = %T, want *value.SassModule", decoded)
	}
	got, err := module.String()
	if err != nil {
		t.Fatal(err)
	}
	if got != `get-module("meta")` {
		t.Errorf("module = %q, want %q", got, `get-module("meta")`)
	}
}
