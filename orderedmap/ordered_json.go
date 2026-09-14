package orderedmap

import (
	"bytes"
	"encoding/json"
	"reflect"
)

// OrderedJSON is a JSON-compatible map that preserves insertion order.
//
// Go's json.Unmarshal into map[string]any produces maps with
// non-deterministic iteration order. Unmarshaling into *OrderedJSON
// preserves the original JSON key order, and recursively applies the
// same ordering to nested objects and arrays of objects.
//
// Matches Dart: LinkedHashMap (which preserves insertion order natively)
type OrderedJSON struct {
	keys []string
	m    map[string]interface{}
}

// Get returns the value for key, or nil if absent.
func (o *OrderedJSON) Get(key string) (interface{}, bool) {
	if o == nil || o.m == nil {
		return nil, false
	}
	v, ok := o.m[key]
	return v, ok
}

// Has returns true if key is present.
func (o *OrderedJSON) Has(key string) bool {
	_, ok := o.Get(key)
	return ok
}

// Keys returns the keys in insertion order.
func (o *OrderedJSON) Keys() []string {
	if o == nil {
		return nil
	}
	return o.keys
}

// Put stores a value for key. If the key is new, it is appended to the
// insertion-order slice.
func (o *OrderedJSON) Put(key string, val interface{}) {
	if o.m == nil {
		o.m = make(map[string]interface{})
	}
	if _, ok := o.m[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.m[key] = val
}

// Len returns the number of entries.
func (o *OrderedJSON) Len() int {
	if o == nil || o.m == nil {
		return 0
	}
	return len(o.m)
}

// NewOrderedJSONFromPairs creates an *OrderedJSON from a list of key-value
// pairs, in insertion order.
func NewOrderedJSONFromPairs(pairs ...Pair[string, interface{}]) *OrderedJSON {
	om := &OrderedJSON{}
	for _, p := range pairs {
		om.Put(p.Key, p.Val)
	}
	return om
}

// UnmarshalJSON implements json.Unmarshaler. It reads the JSON object
// token-by-token to preserve key insertion order, recursively applying
// OrderedJSON to nested objects.
func (o *OrderedJSON) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return &json.UnmarshalTypeError{Value: "object", Type: reflect.TypeOf(o)}
	}

	o.keys = nil
	o.m = make(map[string]interface{})

	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyTok.(string)
		if !ok {
			return &json.UnmarshalTypeError{Value: "string", Type: reflect.TypeOf(keyTok)}
		}

		o.keys = append(o.keys, key)

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return err
		}

		val, err := decodeJSONValue(raw)
		if err != nil {
			return err
		}
		o.m[key] = val
	}
	return nil
}

// decodeJSONValue recursively decodes a JSON value, using *OrderedJSON
// for objects to preserve insertion order.
func decodeJSONValue(raw json.RawMessage) (interface{}, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	switch raw[0] {
	case '{':
		om := &OrderedJSON{}
		if err := om.UnmarshalJSON(raw); err != nil {
			return nil, err
		}
		return om, nil
	case '[':
		var arr []json.RawMessage
		if err := json.Unmarshal(raw, &arr); err != nil {
			return nil, err
		}
		result := make([]interface{}, len(arr))
		for i, r := range arr {
			v, err := decodeJSONValue(r)
			if err != nil {
				return nil, err
			}
			result[i] = v
		}
		return result, nil
	default:
		var v interface{}
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
}
