package param

import "encoding/json"

type status int8

const (
	omitted  status = iota // zero value - not serialized
	null                   // explicitly JSON null
	included               // has a real value
)

// Opt represents an optional parameter of type T with three states:
//   - omitted (zero value): field is absent from JSON
//   - null: field is explicitly JSON null
//   - included: field carries a real value
//
// T is constrained to comparable so the zero-value check (o == Opt[T]{}) functions correctly.
type Opt[T comparable] struct {
	Value  T
	status status
}

// NewOpt creates an Opt with an included value.
func NewOpt[T comparable](v T) Opt[T] {
	return Opt[T]{Value: v, status: included}
}

// Null creates an Opt that marshals as explicit JSON null.
func Null[T comparable]() Opt[T] {
	return Opt[T]{status: null}
}

// Valid reports whether the Opt has a real (included) value.
func (o Opt[T]) Valid() bool { return o.status == included }

// IsNull reports whether the Opt is explicitly null.
func (o Opt[T]) IsNull() bool { return o.status == null }

// IsOmitted reports whether the Opt is in the zero-value (omitted) state.
func (o Opt[T]) IsOmitted() bool { return o == Opt[T]{} }

// Or returns the value if valid, otherwise returns fallback.
func (o Opt[T]) Or(fallback T) T {
	if o.Valid() {
		return o.Value
	}
	return fallback
}

// MarshalJSON implements json.Marshaler.
// Omitted values produce "null", included values produce the JSON encoding of Value.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	if !o.Valid() {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *Opt[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.status = null
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = v
	o.status = included
	return nil
}
