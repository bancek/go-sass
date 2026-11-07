package value

func testNamespace(name string) SelectorNamespace {
	switch name {
	case "*":
		return AnySelectorNS
	case "":
		return NoSelectorNS
	default:
		return NamedNamespace(name)
	}
}
