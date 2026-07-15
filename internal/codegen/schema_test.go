package codegen

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/require"
)

func newField(name string, typ types.Type) *types.Var {
	return types.NewField(token.NoPos, nil, name, typ, false)
}

func newNamedStruct(name string, fields []*types.Var, tags []string) *types.Named {
	st := types.NewStruct(fields, tags)
	return types.NewNamed(types.NewTypeName(token.NoPos, nil, name, nil), st, nil)
}

func anyType() types.Type {
	return types.Universe.Lookup("any").Type()
}

// ---------------------------------------------------------------------------
// Basic types
// ---------------------------------------------------------------------------

func TestGenerateSchema_string(t *testing.T) {
	t.Parallel()
	schema := GenerateSchema(types.Typ[types.String], nil)
	require.Equal(t, map[string]any{"type": "string"}, schema)
}

func TestGenerateSchema_int(t *testing.T) {
	t.Parallel()
	schema := GenerateSchema(types.Typ[types.Int], nil)
	require.Equal(t, map[string]any{"type": "integer"}, schema)
}

func TestGenerateSchema_float64(t *testing.T) {
	t.Parallel()
	schema := GenerateSchema(types.Typ[types.Float64], nil)
	require.Equal(t, map[string]any{"type": "number"}, schema)
}

func TestGenerateSchema_bool(t *testing.T) {
	t.Parallel()
	schema := GenerateSchema(types.Typ[types.Bool], nil)
	require.Equal(t, map[string]any{"type": "boolean"}, schema)
}

// ---------------------------------------------------------------------------
// Containers
// ---------------------------------------------------------------------------

func TestGenerateSchema_sliceString(t *testing.T) {
	t.Parallel()
	sliceType := types.NewSlice(types.Typ[types.String])
	schema := GenerateSchema(sliceType, nil)
	require.Equal(t, map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "string"},
	}, schema)
}

func TestGenerateSchema_mapStringAny(t *testing.T) {
	t.Parallel()
	mapType := types.NewMap(types.Typ[types.String], anyType())
	schema := GenerateSchema(mapType, nil)
	require.Equal(t, map[string]any{
		"type":                 "object",
		"additionalProperties": map[string]any{},
	}, schema)
}

// ---------------------------------------------------------------------------
// Struct variations
// ---------------------------------------------------------------------------

func TestGenerateSchema_structField(t *testing.T) {
	t.Parallel()
	st := newNamedStruct("MyType",
		[]*types.Var{newField("Name", types.Typ[types.String])},
		[]string{`json:"name"`},
	)
	schema := GenerateSchema(st, nil)
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []string{"name"},
	}, schema)
}

func TestGenerateSchema_omitempty(t *testing.T) {
	t.Parallel()
	st := newNamedStruct("MyType",
		[]*types.Var{newField("Name", types.Typ[types.String])},
		[]string{`json:"name,omitempty"`},
	)
	schema := GenerateSchema(st, nil)
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}, schema)
	_, hasRequired := schema["required"]
	require.False(t, hasRequired, "omitempty field should not appear in required")
}

func TestGenerateSchema_pointerField(t *testing.T) {
	t.Parallel()
	st := newNamedStruct("MyType",
		[]*types.Var{newField("Name", types.NewPointer(types.Typ[types.String]))},
		[]string{`json:"name"`},
	)
	schema := GenerateSchema(st, nil)
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}, schema)
	_, hasRequired := schema["required"]
	require.False(t, hasRequired, "pointer field should not appear in required")
}

// ---------------------------------------------------------------------------
// Annotations
// ---------------------------------------------------------------------------

func TestGenerateSchema_enum(t *testing.T) {
	t.Parallel()
	st := newNamedStruct("GetWeatherRequest",
		[]*types.Var{newField("Unit", types.Typ[types.String])},
		[]string{`json:"unit,omitempty"`},
	)
	comments := map[string]string{
		"GetWeatherRequest.Unit": "//tool:enum celsius,fahrenheit",
	}
	schema := GenerateSchema(st, comments)
	props := schema["properties"].(map[string]any)
	unitSchema := props["unit"].(map[string]any)
	require.Equal(t, []string{"celsius", "fahrenheit"}, unitSchema["enum"])
	require.Equal(t, "string", unitSchema["type"])
}

func TestGenerateSchema_docComment(t *testing.T) {
	t.Parallel()
	st := newNamedStruct("GetWeatherRequest",
		[]*types.Var{newField("Location", types.Typ[types.String])},
		[]string{`json:"location"`},
	)
	comments := map[string]string{
		"GetWeatherRequest.Location": "Location is the city or region to get weather for.",
	}
	schema := GenerateSchema(st, comments)
	props := schema["properties"].(map[string]any)
	locSchema := props["location"].(map[string]any)
	require.Equal(t, "Location is the city or region to get weather for.", locSchema["description"])
	require.Equal(t, "string", locSchema["type"])
}

// ---------------------------------------------------------------------------
// Nested struct
// ---------------------------------------------------------------------------

func TestGenerateSchema_nestedStruct(t *testing.T) {
	t.Parallel()
	innerSt := types.NewStruct(
		[]*types.Var{newField("Value", types.Typ[types.String])},
		[]string{`json:"value"`},
	)
	innerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "InnerType", nil),
		innerSt, nil,
	)
	outerSt := types.NewStruct(
		[]*types.Var{newField("Inner", innerNamed)},
		[]string{`json:"inner"`},
	)
	outerNamed := types.NewNamed(
		types.NewTypeName(token.NoPos, nil, "OuterType", nil),
		outerSt, nil,
	)
	schema := GenerateSchema(outerNamed, nil)
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"inner": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"value": map[string]any{"type": "string"},
				},
				"required": []string{"value"},
			},
		},
		"required": []string{"inner"},
	}, schema)
}

// ---------------------------------------------------------------------------
// any / interface{}
// ---------------------------------------------------------------------------

func TestGenerateSchema_any(t *testing.T) {
	t.Parallel()
	schema := GenerateSchema(anyType(), nil)
	require.Equal(t, map[string]any{}, schema)
}

// ---------------------------------------------------------------------------
// snakeCase
// ---------------------------------------------------------------------------

func TestSnakeCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"GetWeather", "get_weather"},
		{"ID", "id"},
		{"UserID", "user_id"},
		{"", ""},
		{"Name", "name"},
		{"XMLParser", "xml_parser"},
		{"getURL", "get_url"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := SnakeCase(tc.input)
			require.Equal(t, tc.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestIsIntegerKind(t *testing.T) {
	t.Parallel()
	require.True(t, isIntegerKind(types.Int))
	require.True(t, isIntegerKind(types.Int64))
	require.True(t, isIntegerKind(types.Uint32))
	require.False(t, isIntegerKind(types.Float64))
	require.False(t, isIntegerKind(types.Bool))
}

func TestExtractEnumFromComment(t *testing.T) {
	t.Parallel()
	enum := extractEnumFromComment("//tool:enum a,b,c")
	require.Equal(t, []string{"a", "b", "c"}, enum)
}

func TestExtractDescFromComment(t *testing.T) {
	t.Parallel()
	desc := extractDescFromComment("Location is the city or region to get weather for.")
	require.Equal(t, "Location is the city or region to get weather for.", desc)
}

func TestExtractDescFromComment_skipsToolDirectives(t *testing.T) {
	t.Parallel()
	comment := "Unit is the temperature unit.\n//tool:enum celsius,fahrenheit"
	desc := extractDescFromComment(comment)
	require.Equal(t, "Unit is the temperature unit.", desc)
}
