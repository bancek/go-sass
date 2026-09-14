package value

import (
	goUrl "net/url"
	"strings"
	"testing"

	"github.com/bancek/go-sass/deprecation"
	"github.com/bancek/go-sass/sasscommon"
)

// ======================================================================
// Helpers
// ======================================================================

func newTestStylesheetParserAtRule(text string) *StylesheetParser {
	p := NewStylesheetParser([]byte(text), nil, false, nil)
	p.indented = false
	p.plainCss = false
	p.currentIndentation = func() int { return 0 }
	p.expectStatementSeparator = func(name string) error {
		if err := p.whitespaceWithoutComments(true); err != nil {
			return err
		}
		if p.scanner.IsDone() {
			return nil
		}
		next := p.scanner.PeekChar(0)
		if next == ';' || next == '}' {
			return nil
		}
		return p.expectChar(';')
	}
	p.atEndOfStatement = func() bool {
		next := p.scanner.PeekChar(0)
		return next < 0 || next == ';' || next == '}' || next == '{'
	}
	p.lookingAtChildren = func() (bool, error) {
		return p.scanner.PeekChar(0) == '{', nil
	}
	p.children = func(child func() (Statement, error)) ([]Statement, error) {
		if p.scanner.PeekChar(0) == '{' {
			p.scanner.ReadChar()
		}
		return []Statement{}, nil
	}
	p.scanElse = func(ifIndentation int) (bool, error) {
		return false, nil
	}
	return p
}

// skipAtRuleName consumes @identifier, returning the start position.
// This simulates atRule having consumed the @name before dispatching to sub-rules.
func skipAtRuleName(p *StylesheetParser) sasscommon.LineScannerState {
	s := p.scanner.State()
	if p.scanner.PeekChar(0) == '@' {
		_ = p.expectChar('@')
		_, _ = p.interpolatedIdentifier()
	}
	return s
}

func urlParse(rawURL string) (*goUrl.URL, error) {
	return goUrl.Parse(rawURL)
}

// ======================================================================
// Group A: parameterList
// ======================================================================

func TestAtRuleParameterListEmpty(t *testing.T) {
	p := newTestStylesheetParserAtRule("()")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 0 {
		t.Errorf("len = %d, want 0", len(params.Parameters))
	}
	if params.RestParameter != nil {
		t.Error("expected no rest parameter")
	}
}

func TestAtRuleParameterListSingleRequired(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 1 {
		t.Fatalf("len = %d, want 1", len(params.Parameters))
	}
	if params.Parameters[0].Name() != "a" {
		t.Errorf("name = %q, want %q", params.Parameters[0].Name(), "a")
	}
	if params.Parameters[0].DefaultValue != nil {
		t.Error("expected no default value")
	}
}

func TestAtRuleParameterListMultiple(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a, $b, $c)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 3 {
		t.Fatalf("len = %d, want 3", len(params.Parameters))
	}
}

func TestAtRuleParameterListWithDefault(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a: 1)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if params.Parameters[0].DefaultValue == nil {
		t.Fatal("expected default value")
	}
	ne, ok := params.Parameters[0].DefaultValue.(*NumberExpression)
	if !ok {
		t.Fatalf("expected NumberExpression, got %T", params.Parameters[0].DefaultValue)
	}
	if ne.Value != 1 {
		t.Errorf("value = %v, want 1", ne.Value)
	}
}

func TestAtRuleParameterListWithDefaultExpression(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a: 1 + 2)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if params.Parameters[0].DefaultValue == nil {
		t.Fatal("expected default value")
	}
	if _, ok := params.Parameters[0].DefaultValue.(*BinaryOperationExpression); !ok {
		t.Errorf("expected BinaryOperationExpression, got %T", params.Parameters[0].DefaultValue)
	}
}

func TestAtRuleParameterListRestParam(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a, $b...)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 1 {
		t.Fatalf("len = %d, want 1", len(params.Parameters))
	}
	if params.RestParameter == nil {
		t.Fatal("expected rest parameter")
	}
	if *params.RestParameter != "b" {
		t.Errorf("rest = %q, want %q", *params.RestParameter, "b")
	}
}

func TestAtRuleParameterListRestTrailingComma(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a...,)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if params.RestParameter == nil {
		t.Fatal("expected rest parameter")
	}
}

func TestAtRuleParameterListDefaultTrailingComma(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a: 1,)")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 1 {
		t.Fatalf("len = %d, want 1", len(params.Parameters))
	}
}

func TestAtRuleParameterListDuplicate(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a, $a)")
	_, err := p.parameterList()
	if err == nil {
		t.Fatal("expected error for duplicate parameter")
	}
	want := strings.Join([]string{
		`Error: Duplicate parameter.`,
		`  ╷`,
		`1 │ ($a, $a)`,
		`  │      ^^`,
		`  ╵`,
		`  - 1:6  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleParameterListMissingCloseParen(t *testing.T) {
	p := newTestStylesheetParserAtRule("($a")
	_, err := p.parameterList()
	if err == nil {
		t.Fatal("expected error for missing )")
	}
}

func TestAtRuleParameterListWhitespace(t *testing.T) {
	p := newTestStylesheetParserAtRule("(  $a , $b  )")
	params, err := p.parameterList()
	if err != nil {
		t.Fatal(err)
	}
	if len(params.Parameters) != 2 {
		t.Fatalf("len = %d, want 2", len(params.Parameters))
	}
}

// ======================================================================
// Group B: configuration
// ======================================================================

func TestAtRuleConfigurationNoWith(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	config, err := p.configuration(false)
	if err != nil {
		t.Fatal(err)
	}
	if config != nil {
		t.Errorf("expected nil, got %d configured vars", len(config))
	}
}

func TestAtRuleConfigurationSimple(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1)")
	config, err := p.configuration(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(config) != 1 {
		t.Fatalf("len = %d, want 1", len(config))
	}
	if config[0].Name() != "a" {
		t.Errorf("name = %q, want %q", config[0].Name(), "a")
	}
	if ne, ok := config[0].Expression.(*NumberExpression); !ok || ne.Value != 1 {
		t.Errorf("expression is not NumberExpression{1}")
	}
	if config[0].IsGuarded {
		t.Error("expected not guarded")
	}
}

func TestAtRuleConfigurationMultiple(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1, $b: red)")
	config, err := p.configuration(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(config) != 2 {
		t.Fatalf("len = %d, want 2", len(config))
	}
}

func TestAtRuleConfigurationGuarded(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1 !default)")
	config, err := p.configuration(true)
	if err != nil {
		t.Fatal(err)
	}
	if !config[0].IsGuarded {
		t.Error("expected guarded=true")
	}
}

func TestAtRuleConfigurationInvalidFlag(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1 !other)")
	_, err := p.configuration(true)
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
	want := strings.Join([]string{
		`Error: Invalid flag name.`,
		`  ╷`,
		`1 │ with ($a: 1 !other)`,
		`  │             ^^^^^^`,
		`  ╵`,
		`  - 1:13  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleConfigurationDuplicate(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1, $a: 2)")
	_, err := p.configuration(false)
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
	want := strings.Join([]string{
		`Error: The same variable may only be configured once.`,
		`  ╷`,
		`1 │ with ($a: 1, $a: 2)`,
		`  │              ^^^^^`,
		`  ╵`,
		`  - 1:14  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleConfigurationTrailingComma(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1,)")
	config, err := p.configuration(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(config) != 1 {
		t.Fatalf("len = %d, want 1", len(config))
	}
}

func TestAtRuleConfigurationNoCloseParen(t *testing.T) {
	p := newTestStylesheetParserAtRule("with ($a: 1")
	_, err := p.configuration(false)
	if err == nil {
		t.Fatal("expected error for missing )")
	}
}

// ======================================================================
// Group C: memberList
// ======================================================================

func TestAtRuleMemberListSingleIdentifier(t *testing.T) {
	p := newTestStylesheetParserAtRule("foo")
	idents, vars, err := p.memberList()
	if err != nil {
		t.Fatal(err)
	}
	if idents.Len() != 1 || !idents.Contains("foo") {
		t.Errorf("idents = %v, want ['foo']", idents.Keys())
	}
	if vars.Len() != 0 {
		t.Errorf("vars len = %d, want 0", vars.Len())
	}
}

func TestAtRuleMemberListSingleVariable(t *testing.T) {
	p := newTestStylesheetParserAtRule("$a")
	idents, vars, err := p.memberList()
	if err != nil {
		t.Fatal(err)
	}
	if idents.Len() != 0 {
		t.Errorf("idents len = %d, want 0", idents.Len())
	}
	if vars.Len() != 1 || !vars.Contains("a") {
		t.Errorf("vars = %v, want ['a']", vars.Keys())
	}
}

func TestAtRuleMemberListMixed(t *testing.T) {
	p := newTestStylesheetParserAtRule("$a, foo, $b")
	idents, vars, err := p.memberList()
	if err != nil {
		t.Fatal(err)
	}
	if idents.Len() != 1 || !idents.Contains("foo") {
		t.Errorf("idents length or content wrong: %v", idents.Keys())
	}
	if vars.Len() != 2 || !vars.Contains("a") || !vars.Contains("b") {
		t.Errorf("vars length or content wrong: %v", vars.Keys())
	}
}

func TestAtRuleMemberListDuplicates(t *testing.T) {
	p := newTestStylesheetParserAtRule("foo, $a, foo")
	idents, vars, err := p.memberList()
	if err != nil {
		t.Fatal(err)
	}
	if idents.Len() != 1 || !idents.Contains("foo") {
		t.Errorf("expected 'foo' exactly once, got %v", idents.Keys())
	}
	if vars.Len() != 1 || !vars.Contains("a") {
		t.Errorf("expected 'a' exactly once, got %v", vars.Keys())
	}
}

// ======================================================================
// Group D: atRootQuery
// ======================================================================

func TestAtRuleAtRootQuerySingleExpression(t *testing.T) {
	p := newTestStylesheetParserAtRule("(foo)")
	interp, err := p.atRootQuery()
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != "(foo)" {
		t.Errorf("AsPlain() = %v, want '(foo)'", s)
	}
}

func TestAtRuleAtRootQueryWithColon(t *testing.T) {
	p := newTestStylesheetParserAtRule("(foo: bar)")
	interp, err := p.atRootQuery()
	if err != nil {
		t.Fatal(err)
	}
	s := interp.AsPlain()
	if s == nil || *s != "(foo: bar)" {
		t.Errorf("AsPlain() = %v, want '(foo: bar)'", s)
	}
}

func TestAtRuleAtRootQueryMissingParen(t *testing.T) {
	p := newTestStylesheetParserAtRule("foo")
	_, err := p.atRootQuery()
	if err == nil {
		t.Fatal("expected error for missing (")
	}
}

// ======================================================================
// Group E: contentRule
// ======================================================================

func TestAtRuleContentRuleNoArgs(t *testing.T) {
	p := newTestStylesheetParserAtRule("@content;")
	p.inMixin = true
	start := skipAtRuleName(p)
	rule, err := p.contentRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Arguments == nil {
		t.Fatal("expected ArgumentList")
	}
}

func TestAtRuleContentRuleWithArgs(t *testing.T) {
	p := newTestStylesheetParserAtRule("@content($a: 1);")
	p.inMixin = true
	start := skipAtRuleName(p)
	rule, err := p.contentRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Arguments.Named.Len() != 1 {
		t.Errorf("named len = %d, want 1", rule.Arguments.Named.Len())
	}
}

func TestAtRuleContentRuleNotInMixin(t *testing.T) {
	p := newTestStylesheetParserAtRule("@content;")
	start := skipAtRuleName(p)
	_, err := p.contentRule(start)
	if err == nil {
		t.Fatal("expected error when not in mixin")
	}
	want := strings.Join([]string{
		`Error: @content is only allowed within mixin declarations.`,
		`  ╷`,
		`1 │ @content;`,
		`  │ ^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group F: debugRule, errorRule, warnRule
// ======================================================================

func TestAtRuleDebugRuleNumber(t *testing.T) {
	p := newTestStylesheetParserAtRule("@debug 42;")
	start := skipAtRuleName(p)
	rule, err := p.debugRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*NumberExpression); !ok {
		t.Errorf("expected NumberExpression, got %T", rule.Expression)
	}
}

func TestAtRuleDebugRuleString(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@debug "hello";`)
	start := skipAtRuleName(p)
	rule, err := p.debugRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*StringExpression); !ok {
		t.Errorf("expected StringExpression, got %T", rule.Expression)
	}
}

func TestAtRuleErrorRuleExpression(t *testing.T) {
	p := newTestStylesheetParserAtRule("@error 1px + 2px;")
	start := skipAtRuleName(p)
	rule, err := p.errorRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*BinaryOperationExpression); !ok {
		t.Errorf("expected BinaryOperationExpression, got %T", rule.Expression)
	}
}

func TestAtRuleErrorRuleVariable(t *testing.T) {
	p := newTestStylesheetParserAtRule("@error $var;")
	start := skipAtRuleName(p)
	rule, err := p.errorRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*VariableExpression); !ok {
		t.Errorf("expected VariableExpression, got %T", rule.Expression)
	}
}

func TestAtRuleWarnRuleString(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@warn "msg";`)
	start := skipAtRuleName(p)
	rule, err := p.warnRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*StringExpression); !ok {
		t.Errorf("expected StringExpression, got %T", rule.Expression)
	}
}

// ======================================================================
// Group G: returnRule
// ======================================================================

func TestAtRuleReturnRuleNumber(t *testing.T) {
	p := newTestStylesheetParserAtRule("@return 42;")
	start := skipAtRuleName(p)
	rule, err := p.returnRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*NumberExpression); !ok {
		t.Errorf("expected NumberExpression, got %T", rule.Expression)
	}
}

func TestAtRuleReturnRuleBoolean(t *testing.T) {
	p := newTestStylesheetParserAtRule("@return true;")
	start := skipAtRuleName(p)
	rule, err := p.returnRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Expression.(*BooleanExpression); !ok {
		t.Errorf("expected BooleanExpression, got %T", rule.Expression)
	}
}

// ======================================================================
// Group H: extendRule
// ======================================================================

func TestAtRuleExtendRuleSimple(t *testing.T) {
	p := newTestStylesheetParserAtRule("@extend .foo;")
	p.inStyleRule = true
	start := skipAtRuleName(p)
	rule, err := p.extendRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.IsOptional {
		t.Error("expected isOptional=false")
	}
}

func TestAtRuleExtendRuleOptional(t *testing.T) {
	p := newTestStylesheetParserAtRule("@extend .foo !optional;")
	p.inStyleRule = true
	start := skipAtRuleName(p)
	rule, err := p.extendRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if !rule.IsOptional {
		t.Error("expected isOptional=true")
	}
}

func TestAtRuleExtendRuleOutsideStyleRule(t *testing.T) {
	p := newTestStylesheetParserAtRule("@extend .foo;")
	start := skipAtRuleName(p)
	_, err := p.extendRule(start)
	if err == nil {
		t.Fatal("expected error outside style rule/mixin/content block")
	}
	want := strings.Join([]string{
		`Error: @extend may only be used within style rules.`,
		`  ╷`,
		`1 │ @extend .foo;`,
		`  │ ^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleUseRuleNotFirst(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@use "path/file";`)
	p.isUseAllowed = false
	start := skipAtRuleName(p)
	_, err := p.useRule(start)
	if err == nil {
		t.Fatal("expected error when use not allowed")
	}
}

// ======================================================================
// Group I: useNamespace
// ======================================================================

func TestAtRuleUseNamespaceExplicit(t *testing.T) {
	p := newTestStylesheetParserAtRule("as foo")
	url, _ := urlParse("http://example.com/_test.scss")
	start := skipAtRuleName(p)
	ns, err := p.useNamespace(url, start)
	if err != nil {
		t.Fatal(err)
	}
	if ns == nil || *ns != "foo" {
		t.Errorf("ns = %v, want 'foo'", ns)
	}
}

func TestAtRuleUseNamespaceStar(t *testing.T) {
	p := newTestStylesheetParserAtRule("as *")
	url, _ := urlParse("http://example.com/_test.scss")
	start := skipAtRuleName(p)
	ns, err := p.useNamespace(url, start)
	if err != nil {
		t.Fatal(err)
	}
	if ns != nil {
		t.Errorf("expected nil for 'as *', got %q", *ns)
	}
}

func TestAtRuleUseNamespaceImplicitUnderscore(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	url, _ := urlParse("http://example.com/_foo.scss")
	start := skipAtRuleName(p)
	ns, err := p.useNamespace(url, start)
	if err != nil {
		t.Fatal(err)
	}
	if ns == nil || *ns != "foo" {
		t.Errorf("ns = %v, want 'foo'", ns)
	}
}

func TestAtRuleUseNamespaceImplicitDeep(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	url, _ := urlParse("http://example.com/a/b/_bar.sass")
	start := skipAtRuleName(p)
	ns, err := p.useNamespace(url, start)
	if err != nil {
		t.Fatal(err)
	}
	if ns == nil || *ns != "bar" {
		t.Errorf("ns = %v, want 'bar'", ns)
	}
}

// ======================================================================
// Group J: useRule
// ======================================================================

func TestAtRuleUseRuleBasic(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@use "path/file";`)
	start := skipAtRuleName(p)
	_, err := p.useRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleUseRuleWithNamespace(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@use "path/file" as ns;`)
	start := skipAtRuleName(p)
	rule, err := p.useRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Namespace == nil || *rule.Namespace != "ns" {
		t.Errorf("namespace = %v, want 'ns'", rule.Namespace)
	}
}

func TestAtRuleUseRuleWithConfig(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@use "path/file" with ($a: 1);`)
	start := skipAtRuleName(p)
	rule, err := p.useRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Configuration) != 1 {
		t.Errorf("config len = %d, want 1", len(rule.Configuration))
	}
}

func TestAtRuleUseRuleWithStarNamespace(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@use "path/file" as *;`)
	start := skipAtRuleName(p)
	rule, err := p.useRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Namespace != nil {
		t.Errorf("expected nil namespace for 'as *', got %q", *rule.Namespace)
	}
}

// ======================================================================
// Group K: forwardRule
// ======================================================================

func TestAtRuleForwardRuleBasic(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file";`)
	start := skipAtRuleName(p)
	_, err := p.forwardRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleForwardRuleShow(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file" show $a, foo;`)
	start := skipAtRuleName(p)
	rule, err := p.forwardRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.ShownMixinsAndFunctions == nil || !rule.ShownMixinsAndFunctions.Contains("foo") {
		t.Error("shownMixinsAndFunctions should contain 'foo'")
	}
	if rule.ShownVariables == nil || !rule.ShownVariables.Contains("a") {
		t.Error("shownVariables should contain 'a'")
	}
}

func TestAtRuleForwardRuleHide(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file" hide foo, $a;`)
	start := skipAtRuleName(p)
	rule, err := p.forwardRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.HiddenMixinsAndFunctions == nil || !rule.HiddenMixinsAndFunctions.Contains("foo") {
		t.Error("hiddenMixinsAndFunctions should contain 'foo'")
	}
	if rule.HiddenVariables == nil || !rule.HiddenVariables.Contains("a") {
		t.Error("hiddenVariables should contain 'a'")
	}
}

func TestAtRuleForwardRulePrefix(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file" as prefix-*;`)
	start := skipAtRuleName(p)
	_, err := p.forwardRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleForwardRuleConfig(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file" with ($a: 1 !default);`)
	start := skipAtRuleName(p)
	rule, err := p.forwardRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Configuration) != 1 {
		t.Fatalf("config len = %d, want 1", len(rule.Configuration))
	}
	if !rule.Configuration[0].IsGuarded {
		t.Error("expected guarded=true")
	}
}

func TestAtRuleForwardRuleNotFirst(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@forward "path/file";`)
	p.isUseAllowed = false
	start := skipAtRuleName(p)
	_, err := p.forwardRule(start)
	if err == nil {
		t.Fatal("expected error when forward not allowed")
	}
}

// ======================================================================
// Group L: isPlainImportUrl
// ======================================================================

func TestAtRuleIsPlainImportUrlCssSuffix(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if !p.isPlainImportUrl("foo.css") {
		t.Error("expected true for .css suffix")
	}
}

func TestAtRuleIsPlainImportUrlHttp(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if !p.isPlainImportUrl("http://x.com/a") {
		t.Error("expected true for http://")
	}
}

func TestAtRuleIsPlainImportUrlHttps(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if !p.isPlainImportUrl("https://x.com/a") {
		t.Error("expected true for https://")
	}
}

func TestAtRuleIsPlainImportUrlDoubleSlash(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if !p.isPlainImportUrl("//example.com/x") {
		t.Error("expected true for // prefix")
	}
}

func TestAtRuleIsPlainImportUrlShort(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if p.isPlainImportUrl("a") {
		t.Error("expected false for short (<5) url")
	}
}

func TestAtRuleIsPlainImportUrlScss(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if p.isPlainImportUrl("foo.scss") {
		t.Error("expected false for .scss suffix")
	}
}

func TestAtRuleIsPlainImportUrlSingleSlash(t *testing.T) {
	p := newTestStylesheetParserAtRule("")
	if p.isPlainImportUrl("/abs/path") {
		t.Error("expected false for single /")
	}
}

// ======================================================================
// Group M: importArgument + importRule
// ======================================================================

func TestAtRuleImportArgumentDynamicScss(t *testing.T) {
	p := newTestStylesheetParserAtRule(`"foo.scss"`)
	imp, err := p.importArgument()
	if err != nil {
		t.Fatal(err)
	}
	di, ok := imp.(*DynamicImport)
	if !ok {
		t.Fatalf("expected DynamicImport, got %T", imp)
	}
	if di.URL().String() != "foo.scss" {
		t.Errorf("url = %q, want %q", di.URL().String(), "foo.scss")
	}
}

func TestAtRuleImportArgumentStaticCss(t *testing.T) {
	p := newTestStylesheetParserAtRule(`"bar.css"`)
	imp, err := p.importArgument()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := imp.(*StaticImport); !ok {
		t.Fatalf("expected StaticImport for .css file, got %T", imp)
	}
}

func TestAtRuleImportArgumentUrlFunction(t *testing.T) {
	p := newTestStylesheetParserAtRule(`url("http://example.com")`)
	imp, err := p.importArgument()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := imp.(*StaticImport); !ok {
		t.Fatalf("expected StaticImport for url(...), got %T", imp)
	}
}

func TestAtRuleImportRuleSingleDynamic(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@import "foo.scss";`)
	start := skipAtRuleName(p)
	rule, err := p.importRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Imports) != 1 {
		t.Fatalf("imports len = %d, want 1", len(rule.Imports))
	}
	if _, ok := rule.Imports[0].(*DynamicImport); !ok {
		t.Errorf("expected DynamicImport, got %T", rule.Imports[0])
	}
}

func TestAtRuleImportRuleCommaSeparated(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@import "a", "b";`)
	start := skipAtRuleName(p)
	rule, err := p.importRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Imports) != 2 {
		t.Fatalf("imports len = %d, want 2", len(rule.Imports))
	}
}

func TestAtRuleImportRuleDeprecation(t *testing.T) {
	p := newTestStylesheetParserAtRule(`@import "foo.scss";`)
	start := skipAtRuleName(p)
	_, err := p.importRule(start)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range p.warnings {
		if w.Deprecation == deprecation.Import {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected IMPORT deprecation warning")
	}
}

// ======================================================================
// Group N: tryImportModifiers
// ======================================================================

func TestAtRuleTryImportModifiersNil(t *testing.T) {
	p := newTestStylesheetParserAtRule(``)
	mods, err := p.tryImportModifiers()
	if err != nil {
		t.Fatal(err)
	}
	if mods != nil {
		t.Error("expected nil modifiers")
	}
}

func TestAtRuleTryImportModifiersIdentifier(t *testing.T) {
	p := newTestStylesheetParserAtRule(`screen`)
	mods, err := p.tryImportModifiers()
	if err != nil {
		t.Fatal(err)
	}
	if mods == nil {
		t.Fatal("expected non-nil modifiers")
	}
}

func TestAtRuleTryImportModifiersMediaQuery(t *testing.T) {
	p := newTestStylesheetParserAtRule(`(min-width: 400px)`)
	mods, err := p.tryImportModifiers()
	if err != nil {
		t.Fatal(err)
	}
	if mods == nil {
		t.Fatal("expected non-nil modifiers")
	}
}

// ======================================================================
// Group O: mixinRule
// ======================================================================

func TestAtRuleMixinRuleSimple(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin foo")
	start := skipAtRuleName(p)
	rule, err := p.mixinRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Name() != "foo" {
		t.Errorf("name = %q, want %q", rule.Name(), "foo")
	}
}

func TestAtRuleMixinRuleWithParams(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin foo($a: 1)")
	start := skipAtRuleName(p)
	_, err := p.mixinRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleMixinRuleDoubleDash(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin --bad")
	start := skipAtRuleName(p)
	_, err := p.mixinRule(start)
	if err == nil {
		t.Fatal("expected error for -- prefix")
	}
	want := strings.Join([]string{
		`Error: Sass @mixin names beginning with -- are forbidden for forward-compatibility with plain CSS mixins.`,
		``,
		`For details, see https://sass-lang.com/d/css-function-mixin`,
		`  ╷`,
		`1 │ @mixin --bad`,
		`  │        ^^^^^`,
		`  ╵`,
		`  - 1:8  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleMixinRuleNestedInMixin(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin bar")
	p.inMixin = true
	start := skipAtRuleName(p)
	_, err := p.mixinRule(start)
	if err == nil {
		t.Fatal("expected error for mixin in mixin")
	}
	want := strings.Join([]string{
		`Error: Mixins may not contain mixin declarations.`,
		`  ╷`,
		`1 │ @mixin bar`,
		`  │ ^^^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleMixinRuleInControlDirective(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin bar")
	p.inControlDirective = true
	start := skipAtRuleName(p)
	_, err := p.mixinRule(start)
	if err == nil {
		t.Fatal("expected error for mixin in control directive")
	}
	want := strings.Join([]string{
		`Error: Mixins may not be declared in control directives.`,
		`  ╷`,
		`1 │ @mixin bar`,
		`  │ ^^^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group P: functionRule
// ======================================================================

func TestAtRuleFunctionRuleSimple(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function foo()")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("foo", p.scanner.SpanFromTo(10, 13))
	stmt, err := p.functionRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
	fr, ok := stmt.(*FunctionRule)
	if !ok {
		t.Fatalf("expected FunctionRule, got %T", stmt)
	}
	if fr.Name() != "foo" {
		t.Errorf("name = %q, want %q", fr.Name(), "foo")
	}
}

func TestAtRuleFunctionRuleWithParams(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function foo($a: 1)")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("foo", p.scanner.SpanFromTo(10, 13))
	_, err := p.functionRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleFunctionRuleReservedType(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function type()")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("type", p.scanner.SpanFromTo(10, 14))
	_, err := p.functionRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for reserved name 'type'")
	}
	if err == nil {
		t.Fatal("unexpected")
	}
}

func TestAtRuleFunctionRuleInvalidExpression(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function expression()")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("expression", p.scanner.SpanFromTo(10, 20))
	_, err := p.functionRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for invalid name 'expression'")
	}
	if err == nil {
		t.Fatal("unexpected")
	}
}

func TestAtRuleFunctionRuleInvalidAnd(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function and()")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("and", p.scanner.SpanFromTo(10, 13))
	_, err := p.functionRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for invalid name 'and'")
	}
}

func TestAtRuleFunctionRuleNestedInMixin(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function foo()")
	p.inMixin = true
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("foo", p.scanner.SpanFromTo(10, 13))
	_, err := p.functionRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for function in mixin")
	}
	want := strings.Join([]string{
		`Error: Mixins may not contain function declarations.`,
		`  ╷`,
		`1 │ @function foo()`,
		`  │ ^^^^^^^^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestAtRuleFunctionRuleInControlDirective(t *testing.T) {
	p := newTestStylesheetParserAtRule("@function foo()")
	p.inControlDirective = true
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("foo", p.scanner.SpanFromTo(10, 13))
	_, err := p.functionRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for function in control directive")
	}
	want := strings.Join([]string{
		`Error: Functions may not be declared in control directives.`,
		`  ╷`,
		`1 │ @function foo()`,
		`  │ ^^^^^^^^^^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group Q: eachRule
// ======================================================================

func TestAtRuleEachRuleSingleVar(t *testing.T) {
	p := newTestStylesheetParserAtRule("@each $item in $list")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.eachRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Variables) != 1 || rule.Variables[0] != "item" {
		t.Errorf("variables = %v, want ['item']", rule.Variables)
	}
}

func TestAtRuleEachRuleMultipleVars(t *testing.T) {
	p := newTestStylesheetParserAtRule("@each $k, $v in $map")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.eachRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Variables) != 2 {
		t.Errorf("variables len = %d, want 2", len(rule.Variables))
	}
}

// ======================================================================
// Group R: forRule
// ======================================================================

func TestAtRuleForRuleThrough(t *testing.T) {
	p := newTestStylesheetParserAtRule("@for $i from 1 through 5")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.forRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Variable != "i" {
		t.Errorf("variable = %q, want %q", rule.Variable, "i")
	}
	if rule.IsExclusive {
		t.Error("expected isExclusive=false for 'through'")
	}
}

func TestAtRuleForRuleTo(t *testing.T) {
	p := newTestStylesheetParserAtRule("@for $i from 1 to 5")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.forRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if !rule.IsExclusive {
		t.Error("expected isExclusive=true for 'to'")
	}
}

func TestAtRuleForRuleMissingToOrThrough(t *testing.T) {
	p := newTestStylesheetParserAtRule("@for $i from 1")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	_, err := p.forRule(start, child)
	if err == nil {
		t.Fatal("expected error for missing to/through")
	}
}

// ======================================================================
// Group S: ifRule, whileRule
// ======================================================================

func TestAtRuleIfRuleCondition(t *testing.T) {
	p := newTestStylesheetParserAtRule("@if $x")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.ifRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.Clauses) != 1 {
		t.Fatalf("clauses len = %d, want 1", len(rule.Clauses))
	}
}

func TestAtRuleWhileRuleCondition(t *testing.T) {
	p := newTestStylesheetParserAtRule("@while $x > 0")
	start := skipAtRuleName(p)
	child := func() (Statement, error) { return nil, nil }
	rule, err := p.whileRule(start, child)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rule.Condition.(*BinaryOperationExpression); !ok {
		t.Errorf("expected BinaryOperationExpression, got %T", rule.Condition)
	}
}

// ======================================================================
// Group T: mediaRule, supportsRule, includeRule
// ======================================================================

func TestAtRuleMediaRuleSimple(t *testing.T) {
	p := newTestStylesheetParserAtRule("@media screen")
	start := skipAtRuleName(p)
	_, err := p.mediaRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleSupportsRuleHeader(t *testing.T) {
	p := newTestStylesheetParserAtRule("@supports (display: grid)")
	start := skipAtRuleName(p)
	_, err := p.supportsRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleIncludeRuleNoArgs(t *testing.T) {
	p := newTestStylesheetParserAtRule("@include mixin;")
	start := skipAtRuleName(p)
	_, err := p.includeRule(start)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleIncludeRuleNamespace(t *testing.T) {
	p := newTestStylesheetParserAtRule("@include ns.mixin-name;")
	start := skipAtRuleName(p)
	rule, err := p.includeRule(start)
	if err != nil {
		t.Fatal(err)
	}
	if rule.Namespace() == nil || *rule.Namespace() != "ns" {
		t.Errorf("namespace = %v, want 'ns'", rule.Namespace())
	}
}

// ======================================================================
// Group U: mozDocumentRule
// ======================================================================

func TestAtRuleMozDocumentUrl(t *testing.T) {
	p := newTestStylesheetParserAtRule("@-moz-document url(http://example.com)")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("-moz-document", p.scanner.SpanFromTo(1, 14))
	_, err := p.mozDocumentRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleMozDocumentUrlPrefix(t *testing.T) {
	p := newTestStylesheetParserAtRule("@-moz-document url-prefix(foo)")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("-moz-document", p.scanner.SpanFromTo(1, 14))
	_, err := p.mozDocumentRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleMozDocumentInvalidFunc(t *testing.T) {
	p := newTestStylesheetParserAtRule("@-moz-document invalid()")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("-moz-document", p.scanner.SpanFromTo(1, 14))
	_, err := p.mozDocumentRule(start, nameInterp)
	if err == nil {
		t.Fatal("expected error for invalid function")
	}
	want := strings.Join([]string{
		`Error: Invalid function name.`,
		`  ╷`,
		`1 │ @-moz-document invalid()`,
		`  │                ^^^^^^^`,
		`  ╵`,
		`  - 1:16  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group V: disallowedAtRule
// ======================================================================

func TestAtRuleDisallowedAtRule(t *testing.T) {
	p := newTestStylesheetParserAtRule("@else { }")
	start := skipAtRuleName(p)
	_, err := p.disallowedAtRule(start)
	if err == nil {
		t.Fatal("expected error for disallowed at-rule")
	}
	want := strings.Join([]string{
		`Error: This at-rule is not allowed here.`,
		`  ╷`,
		`1 │ @else { }`,
		`  │ ^^^^^^`,
		`  ╵`,
		`  - 1:1  root stylesheet`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// ======================================================================
// Group W: unknownAtRule
// ======================================================================

func TestAtRuleUnknownAtRuleWithValue(t *testing.T) {
	p := newTestStylesheetParserAtRule("@custom-rule value;")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("custom-rule", p.scanner.SpanFromTo(1, 12))
	_, err := p.unknownAtRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAtRuleUnknownAtRuleNoValue(t *testing.T) {
	p := newTestStylesheetParserAtRule("@custom;")
	start := skipAtRuleName(p)
	nameInterp := NewInterpolationPlain("custom", p.scanner.SpanFromTo(1, 7))
	_, err := p.unknownAtRule(start, nameInterp)
	if err != nil {
		t.Fatal(err)
	}
}

// ======================================================================
// Group X: plainAtRuleName
// ======================================================================

func TestAtRulePlainAtRuleName(t *testing.T) {
	p := newTestStylesheetParserAtRule("@mixin")
	name, err := p.plainAtRuleName()
	if err != nil {
		t.Fatal(err)
	}
	if name != "mixin" {
		t.Errorf("name = %q, want %q", name, "mixin")
	}
}

func TestAtRulePlainAtRuleNameNoAt(t *testing.T) {
	p := newTestStylesheetParserAtRule("mixin")
	_, err := p.plainAtRuleName()
	if err == nil {
		t.Fatal("expected error for missing @")
	}
}

// ======================================================================
// Group Y: functionChild
// ======================================================================

func TestAtRuleFunctionChildDebug(t *testing.T) {
	p := newTestStylesheetParserAtRule("@debug 1;")
	stmt, err := p.functionChild()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stmt.(*DebugRule); !ok {
		t.Errorf("expected DebugRule, got %T", stmt)
	}
}

func TestAtRuleFunctionChildReturn(t *testing.T) {
	p := newTestStylesheetParserAtRule("@return $x;")
	stmt, err := p.functionChild()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := stmt.(*ReturnRule); !ok {
		t.Errorf("expected ReturnRule, got %T", stmt)
	}
}

func TestAtRuleFunctionChildDisallowed(t *testing.T) {
	p := newTestStylesheetParserAtRule("@else")
	_, err := p.functionChild()
	if err == nil {
		t.Fatal("expected error for disallowed at-rule in function")
	}
}
