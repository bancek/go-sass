// Copyright 2019 Google LLC. Use of this source code is governed by an
// MIT-style license that can be found in the LICENSE file or at
// https://opensource.org/licenses/MIT.
//
// Ported and rearchitected for Go by Luka Zakrajsek.

package configuration

// dart-source: lib/src/configuration.dart

import (
	"strings"

	"github.com/bancek/go-sass/sasscommon"
	"github.com/bancek/go-sass/value"
)

// valueStore is the underlying storage for configuration values, supporting
// lookup and removal similar to Dart's Map views.
//
// Removal matters: consuming a variable marks it used by removing it, even
// when the underlying map sits behind prefix/show/hide wrappers. Each
// wrapper below mirrors one of Dart's Map views (UnprefixedMapView,
// LimitedMapView safelist/blocklist) with Remove support preserved.
type valueStore interface {
	Get(name string) (*ConfiguredValue, bool)
	Remove(name string) *ConfiguredValue
	Len() int
	Keys() []string
}

// directStore holds values in a plain map: the base layer every
// configuration starts from.
//
// Matches Dart: the raw _values map
type directStore struct {
	values map[string]*ConfiguredValue
}

func (s *directStore) Get(name string) (*ConfiguredValue, bool) {
	v, ok := s.values[name]
	return v, ok
}

func (s *directStore) Remove(name string) *ConfiguredValue {
	v := s.values[name]
	delete(s.values, name)
	return v
}

func (s *directStore) Len() int {
	return len(s.values)
}

func (s *directStore) Keys() []string {
	keys := make([]string, 0, len(s.values))
	for k := range s.values {
		keys = append(keys, k)
	}
	return keys
}

// unprefixedStore wraps a store stripping prefix from keys, so a @forward
// with a prefix only exposes the configuration variables visible under it.
// Removal forwards with the prefix re-attached, marking use in the inner map.
//
// Matches Dart: UnprefixedMapView
type unprefixedStore struct {
	inner  valueStore
	prefix string
}

func (s *unprefixedStore) Get(name string) (*ConfiguredValue, bool) {
	return s.inner.Get(s.prefix + name)
}

func (s *unprefixedStore) Remove(name string) *ConfiguredValue {
	return s.inner.Remove(s.prefix + name)
}

func (s *unprefixedStore) Len() int {
	return len(s.Keys())
}

func (s *unprefixedStore) Keys() []string {
	var keys []string
	for _, k := range s.inner.Keys() {
		if strings.HasPrefix(k, s.prefix) {
			keys = append(keys, k[len(s.prefix):])
		}
	}
	return keys
}

// safelistStore wraps a store allowing only the @forward show-list keys
// through: hidden names read as absent and remove as a no-op.
//
// Matches Dart: LimitedMapView.safelist
type safelistStore struct {
	inner    valueStore
	safelist map[string]struct{}
}

func (s *safelistStore) Get(name string) (*ConfiguredValue, bool) {
	if _, ok := s.safelist[name]; !ok {
		return nil, false
	}
	return s.inner.Get(name)
}

func (s *safelistStore) Remove(name string) *ConfiguredValue {
	if _, ok := s.safelist[name]; !ok {
		return nil
	}
	return s.inner.Remove(name)
}

func (s *safelistStore) Len() int {
	return len(s.Keys())
}

func (s *safelistStore) Keys() []string {
	var keys []string
	for _, k := range s.inner.Keys() {
		if _, ok := s.safelist[k]; ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// blocklistStore wraps a store hiding the @forward hide-list keys: blocked
// names read as absent and remove as a no-op, everything else passes through.
//
// Matches Dart: LimitedMapView.blocklist
type blocklistStore struct {
	inner     valueStore
	blocklist map[string]struct{}
}

func (s *blocklistStore) Get(name string) (*ConfiguredValue, bool) {
	if _, ok := s.blocklist[name]; ok {
		return nil, false
	}
	return s.inner.Get(name)
}

func (s *blocklistStore) Remove(name string) *ConfiguredValue {
	if _, ok := s.blocklist[name]; ok {
		return nil
	}
	return s.inner.Remove(name)
}

func (s *blocklistStore) Len() int {
	return len(s.Keys())
}

func (s *blocklistStore) Keys() []string {
	var keys []string
	for _, k := range s.inner.Keys() {
		if _, ok := s.blocklist[k]; !ok {
			keys = append(keys, k)
		}
	}
	return keys
}

// Configuration is a set of variables meant to configure a module by
// overriding its !default declarations.
//
// A configuration may be either implicit, meaning that it's either empty or
// created by importing a file containing a @forward rule; or explicit, meaning
// that it's created by passing a `with` clause to a @use rule. Explicit
// configurations have spans associated with them and are represented by the
// ExplicitConfiguration type.
type Configuration struct {
	store                 valueStore
	originalConfiguration *Configuration
	isExplicit            bool
	// NodeWithSpan is the node whose span marks where an explicit
	// configuration was declared. It is nil for implicit configurations,
	// which carry no `with` clause to point at.
	NodeWithSpan sasscommon.AstNode
}

// IsExplicit returns whether this Configuration was created as an
// ExplicitConfiguration.
func (c *Configuration) IsExplicit() bool {
	return c.isExplicit
}

// newConfiguration creates a Configuration with the given store and original.
//
// The original links copies made through @forward back to their base, so
// SameOriginal can tell configurations split from one `with` clause apart
// from independently created ones.
func newConfiguration(store valueStore, original *Configuration) *Configuration {
	return &Configuration{
		store:                 store,
		originalConfiguration: original,
	}
}

// NewConfigurationImplicit creates an implicit configuration with the given values.
//
// Implicit configurations come from @import-era variable state rather than a
// `with` clause, so they are silently ignored when the module already loaded.
//
// Matches Dart: Configuration.implicit
func NewConfigurationImplicit(values map[string]*ConfiguredValue) *Configuration {
	return &Configuration{
		store: &directStore{values: values},
	}
}

// The backing value for originalConfiguration.
//
// This is nil if originalConfig() refers to itself since this can't be
// assigned to a pointer field during initialization.
func (c *Configuration) originalConfig() *Configuration {
	if c.originalConfiguration != nil {
		return c.originalConfiguration
	}
	return c
}

// SameOriginal returns whether this and that Configuration have the same
// original configuration.
//
// An implicit configuration will always return false because it was not
// created through another configuration.
//
// ExplicitConfigurations and configurations created through ThroughForward
// will be considered to have the same original config if they were created as
// a copy from the same base configuration.
func (c *Configuration) SameOriginal(that *Configuration) bool {
	return c.originalConfig() == that.originalConfig()
}

// EmptyConfiguration returns the empty configuration, which indicates that the
// module has not been configured.
//
// Empty configurations are always considered implicit, since they are ignored
// if the module has already been loaded.
func EmptyConfiguration() *Configuration {
	return &Configuration{
		store: &directStore{values: make(map[string]*ConfiguredValue)},
	}
}

// IsEmpty returns whether this configuration has no variables.
func (c *Configuration) IsEmpty() bool {
	return c.store.Len() == 0
}

// Get returns the configured value for name, or false if not present.
//
// Lookups pass through any @forward wrappers, so only variables visible
// through the forwards that produced this configuration resolve.
func (c *Configuration) Get(name string) (*ConfiguredValue, bool) {
	return c.store.Get(name)
}

// Values returns a copy of the values map.
//
// The copy is snapshot through the wrappers, so callers see exactly the
// variables visible through this configuration without aliasing its store.
func (c *Configuration) Values() map[string]*ConfiguredValue {
	values := make(map[string]*ConfiguredValue)
	for _, k := range c.store.Keys() {
		if v, ok := c.store.Get(k); ok {
			values[k] = v
		}
	}
	return values
}

// Remove removes a variable with name from this configuration, returning it.
//
// If no such variable exists in this configuration, returns nil.
func (c *Configuration) Remove(name string) *ConfiguredValue {
	if c.IsEmpty() {
		return nil
	}
	return c.store.Remove(name)
}

// ThroughForward creates a new configuration from this one based on a
// @forward rule.
//
// Only allow variables that are visible through the @forward to be configured.
// These views support Remove so we can mark when a configuration variable is
// used by removing it even when the underlying map is wrapped.
func (c *Configuration) ThroughForward(forward *value.ForwardRule) *Configuration {
	if c.IsEmpty() {
		return c
	}

	var store valueStore = c.store

	// Layer the @forward visibility: prefix first, then the show/hide
	// lists. Each wrapper supports Remove so consuming a variable marks it
	// used in the underlying map.
	if forward.Prefix != nil {
		store = &unprefixedStore{inner: store, prefix: *forward.Prefix}
	}

	if forward.ShownVariables != nil {
		store = &safelistStore{inner: store, safelist: forward.ShownVariables.Map()}
	} else if forward.HiddenVariables != nil && forward.HiddenVariables.Len() > 0 {
		store = &blocklistStore{inner: store, blocklist: forward.HiddenVariables.Map()}
	}

	return &Configuration{
		store:                 store,
		originalConfiguration: c.originalConfig(),
		isExplicit:            c.isExplicit,
		NodeWithSpan:          c.NodeWithSpan,
	}
}

// String renders the configuration in Sass map form, ($name: value, ...),
// delegating each value to its own string form.
func (c *Configuration) String() (string, error) {
	var parts []string
	for _, k := range c.store.Keys() {
		if v, ok := c.store.Get(k); ok {
			s, err := v.String()
			if err != nil {
				return "", err
			}
			parts = append(parts, "$"+k+": "+s)
		}
	}
	return "(" + strings.Join(parts, ",") + ")", nil
}

// ExplicitConfiguration is a Configuration that was created with an explicit
// `with` clause of a @use rule.
//
// Both types of configuration pass through @forward rules, but explicit
// configurations will cause an error if attempting to use them on a module
// that has already been loaded, while implicit configurations will be silently
// ignored in this case.
type ExplicitConfiguration struct {
	*Configuration
}

// NewExplicitConfiguration creates an ExplicitConfiguration with a values map
// and a nodeWithSpan.
func NewExplicitConfiguration(values map[string]*ConfiguredValue, nodeWithSpan sasscommon.AstNode) *ExplicitConfiguration {
	return &ExplicitConfiguration{
		Configuration: &Configuration{
			store:        &directStore{values: values},
			isExplicit:   true,
			NodeWithSpan: nodeWithSpan,
		},
	}
}

// NewExplicitConfigurationWithOriginal creates an ExplicitConfiguration with a
// values map, a nodeWithSpan, and a reference to the original configuration.
func NewExplicitConfigurationWithOriginal(values map[string]*ConfiguredValue, nodeWithSpan sasscommon.AstNode, originalConfiguration *Configuration) *ExplicitConfiguration {
	return &ExplicitConfiguration{
		Configuration: &Configuration{
			store:                 &directStore{values: values},
			originalConfiguration: originalConfiguration,
			isExplicit:            true,
			NodeWithSpan:          nodeWithSpan,
		},
	}
}

// ThroughForward returns a new ExplicitConfiguration based on this one with
// the given @forward rule applied, preserving the explicit nature and node span.
func (c *ExplicitConfiguration) ThroughForward(forward *value.ForwardRule) *ExplicitConfiguration {
	result := c.Configuration.ThroughForward(forward)
	if result == c.Configuration {
		return c
	}
	return &ExplicitConfiguration{
		Configuration: result,
	}
}
