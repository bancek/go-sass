package spec

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SpecOptions represents parsed options.yml content for a spec directory.
type SpecOptions struct {
	Todo        []string `yaml:":todo"`
	WarningTodo []string `yaml:":warning_todo"`
	IgnoreFor   []string `yaml:":ignore_for"`
	Precision   int      `yaml:":precision"`
}

// ParseOptions parses YAML content into SpecOptions. Returns empty options
// if content is empty or whitespace-only.
func ParseOptions(content string) (*SpecOptions, error) {
	content = trimSpace(content)
	if content == "" {
		return &SpecOptions{}, nil
	}
	var opts SpecOptions
	if err := yaml.Unmarshal([]byte(content), &opts); err != nil {
		return nil, fmt.Errorf("parsing options.yml: %w", err)
	}
	return &opts, nil
}

// Merge merges child options into parent, with child taking precedence.
func (o *SpecOptions) Merge(child *SpecOptions) *SpecOptions {
	if child == nil {
		return o
	}
	if o == nil {
		return child
	}
	result := &SpecOptions{
		Precision: o.Precision,
	}
	// Child options take precedence.
	if child.Precision != 0 {
		result.Precision = child.Precision
	}
	// For lists, child completely overrides parent at the list level
	// (matching Dart behavior: options.yml in a subdirectory completely
	// overrides the parent's value for each key).
	if len(child.Todo) > 0 {
		result.Todo = child.Todo
	} else {
		result.Todo = o.Todo
	}
	if len(child.WarningTodo) > 0 {
		result.WarningTodo = child.WarningTodo
	} else {
		result.WarningTodo = o.WarningTodo
	}
	if len(child.IgnoreFor) > 0 {
		result.IgnoreFor = child.IgnoreFor
	} else {
		result.IgnoreFor = o.IgnoreFor
	}
	return result
}

// IsTodo reports whether the given implementation is marked as :todo.
func (o *SpecOptions) IsTodo(impl string) bool {
	return containsString(o.Todo, impl)
}

// IsTodoAny reports whether any of the given implementations is marked as :todo.
func (o *SpecOptions) IsTodoAny(impls []string) bool {
	for _, impl := range impls {
		if containsString(o.Todo, impl) {
			return true
		}
	}
	return false
}

// IsWarningTodo reports whether the given implementation is marked as :warning_todo.
func (o *SpecOptions) IsWarningTodo(impl string) bool {
	return containsString(o.WarningTodo, impl)
}

// IsWarningTodoAny reports whether any of the given implementations is marked
// as :warning_todo.
func (o *SpecOptions) IsWarningTodoAny(impls []string) bool {
	for _, impl := range impls {
		if containsString(o.WarningTodo, impl) {
			return true
		}
	}
	return false
}

// IsIgnored reports whether the given implementation is marked as :ignore_for.
func (o *SpecOptions) IsIgnored(impl string) bool {
	return containsString(o.IgnoreFor, impl)
}

// IsIgnoredAny reports whether any of the given implementations is marked
// as :ignore_for.
func (o *SpecOptions) IsIgnoredAny(impls []string) bool {
	for _, impl := range impls {
		if containsString(o.IgnoreFor, impl) {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if strings.Contains(item, s) {
			return true
		}
	}
	return false
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
