package param

import (
	"encoding/json"
	"testing"
)

type myCustomType string

func TestNewOpt_createsValidOptWithCorrectValue(t *testing.T) {
	t.Parallel()

	opt := NewOpt(42)
	if !opt.Valid() {
		t.Error("expected Valid() to be true for NewOpt")
	}
	if opt.IsNull() {
		t.Error("expected IsNull() to be false for NewOpt")
	}
	if opt.IsOmitted() {
		t.Error("expected IsOmitted() to be false for NewOpt")
	}
	if opt.Value != 42 {
		t.Errorf("expected Value to be 42, got %d", opt.Value)
	}
}

func TestNull_createsNullOpt(t *testing.T) {
	t.Parallel()

	opt := Null[string]()
	if opt.IsNull() != true {
		t.Error("expected IsNull() to be true for Null")
	}
	if opt.Valid() {
		t.Error("expected Valid() to be false for Null")
	}
	if opt.IsOmitted() {
		t.Error("expected IsOmitted() to be false for Null")
	}
}

func TestZeroValue_isOmitted(t *testing.T) {
	t.Parallel()

	var opt Opt[int]
	if !opt.IsOmitted() {
		t.Error("expected zero value Opt to be omitted")
	}
	if opt.Valid() {
		t.Error("expected zero value Opt to not be valid")
	}
	if opt.IsNull() {
		t.Error("expected zero value Opt to not be null")
	}
}

func TestOr_returnsValueWhenValid(t *testing.T) {
	t.Parallel()

	opt := NewOpt("hello")
	result := opt.Or("fallback")
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestOr_returnsFallbackWhenNotValid(t *testing.T) {
	t.Parallel()

	var opt Opt[string]
	result := opt.Or("fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got %q", result)
	}
}

func TestOr_returnsFallbackWhenNull(t *testing.T) {
	t.Parallel()

	opt := Null[string]()
	result := opt.Or("fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got %q", result)
	}
}

func TestMarshalJSON_included(t *testing.T) {
	t.Parallel()

	opt := NewOpt(42)
	data, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "42" {
		t.Errorf("expected '42', got %s", data)
	}
}

func TestMarshalJSON_null(t *testing.T) {
	t.Parallel()

	opt := Null[int]()
	data, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("expected 'null', got %s", data)
	}
}

func TestMarshalJSON_omitted(t *testing.T) {
	t.Parallel()

	var opt Opt[int]
	data, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("expected 'null', got %s", data)
	}
}

func TestUnmarshalJSON_null(t *testing.T) {
	t.Parallel()

	var opt Opt[string]
	if err := json.Unmarshal([]byte("null"), &opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opt.IsNull() {
		t.Error("expected opt to be null after unmarshaling null")
	}
	if opt.Valid() {
		t.Error("expected opt to not be valid after unmarshaling null")
	}
}

func TestUnmarshalJSON_value(t *testing.T) {
	t.Parallel()

	var opt Opt[int]
	if err := json.Unmarshal([]byte("42"), &opt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opt.Valid() {
		t.Error("expected opt to be valid after unmarshaling value")
	}
	if opt.Value != 42 {
		t.Errorf("expected Value to be 42, got %d", opt.Value)
	}
	if opt.IsNull() {
		t.Error("expected opt to not be null after unmarshaling value")
	}
}

func TestInt_convenience(t *testing.T) {
	t.Parallel()

	opt := Int(1024)
	if !opt.Valid() {
		t.Error("expected Int() to create valid Opt")
	}
	if opt.Value != 1024 {
		t.Errorf("expected 1024, got %d", opt.Value)
	}
}

func TestFloat_convenience(t *testing.T) {
	t.Parallel()

	opt := Float(3.14)
	if !opt.Valid() {
		t.Error("expected Float() to create valid Opt")
	}
	if opt.Value != 3.14 {
		t.Errorf("expected 3.14, got %f", opt.Value)
	}
}

func TestBool_convenience(t *testing.T) {
	t.Parallel()

	opt := Bool(true)
	if !opt.Valid() {
		t.Error("expected Bool() to create valid Opt")
	}
	if opt.Value != true {
		t.Errorf("expected true, got %v", opt.Value)
	}
}

func TestString_convenience(t *testing.T) {
	t.Parallel()

	opt := String("hello")
	if !opt.Valid() {
		t.Error("expected String() to create valid Opt")
	}
	if opt.Value != "hello" {
		t.Errorf("expected 'hello', got %q", opt.Value)
	}
}

func TestCustomComparableType(t *testing.T) {
	t.Parallel()

	const expected myCustomType = "custom-value"
	opt := NewOpt(expected)
	if !opt.Valid() {
		t.Error("expected Valid() to be true for custom type")
	}
	if opt.Value != expected {
		t.Errorf("expected %q, got %q", expected, opt.Value)
	}
}

func TestRoundTrip_marshalThenUnmarshal(t *testing.T) {
	t.Parallel()

	// Round-trip with a value
	original := NewOpt("round-trip")
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Opt[string]
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.Valid() {
		t.Error("expected decoded value to be valid")
	}
	if decoded.Value != "round-trip" {
		t.Errorf("expected 'round-trip', got %q", decoded.Value)
	}

	// Round-trip with null
	nullOpt := Null[string]()
	nullData, err := json.Marshal(nullOpt)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decodedNull Opt[string]
	if err := json.Unmarshal(nullData, &decodedNull); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decodedNull.IsNull() {
		t.Error("expected decoded null to be null")
	}
	if decodedNull.Valid() {
		t.Error("expected decoded null to not be valid")
	}
}
