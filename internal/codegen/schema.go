package codegen

import (
	"go/types"
	"reflect"
	"strings"
	"unicode"
)

// GenerateSchema produces a JSON Schema map[string]any from a go/types Type.
// It handles: string, int/uint variants, float variants, bool, slices, maps,
// structs, pointers, any.
// For structs: reads json tags for field names and omitempty,
// reads doc comments from the AST for field descriptions,
// reads //tool:enum annotations for enum values.
func GenerateSchema(typ types.Type, comments map[string]string) map[string]any {
	if typ == nil {
		return map[string]any{}
	}

	switch t := typ.(type) {
	case *types.Basic:
		return generateBasicSchema(t)
	case *types.Slice:
		return map[string]any{
			"type":  "array",
			"items": GenerateSchema(t.Elem(), comments),
		}
	case *types.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": GenerateSchema(t.Elem(), comments),
		}
	case *types.Pointer:
		return GenerateSchema(t.Elem(), comments)
	case *types.Named:
		switch u := t.Underlying().(type) {
		case *types.Struct:
			return generateStructSchema(u, t.Obj().Name(), comments)
		case *types.Interface:
			return map[string]any{}
		default:
			return GenerateSchema(u, comments)
		}
	case *types.Struct:
		return generateStructSchema(t, "", comments)
	case *types.Interface:
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func generateBasicSchema(b *types.Basic) map[string]any {
	switch {
	case isIntegerKind(b.Kind()):
		return map[string]any{"type": "integer"}
	case b.Kind() == types.Float32 || b.Kind() == types.Float64:
		return map[string]any{"type": "number"}
	case b.Kind() == types.Bool:
		return map[string]any{"type": "boolean"}
	case b.Kind() == types.String:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{}
	}
}

func generateStructSchema(st *types.Struct, typeName string, comments map[string]string) map[string]any {
	props := make(map[string]any)
	var required []string

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}

		jsonName, omitempty := parseJSONTag(st.Tag(i))
		if jsonName == "-" {
			continue
		}
		if jsonName == "" {
			jsonName = field.Name()
		}

		fieldType := field.Type()
		_, isPtr := fieldType.(*types.Pointer)

		fieldSchema := GenerateSchema(fieldType, comments)

		if typeName != "" {
			commentKey := typeName + "." + field.Name()
			if comment, ok := comments[commentKey]; ok {
				if enum := extractEnumFromComment(comment); len(enum) > 0 {
					fieldSchema["enum"] = enum
				}
				if desc := extractDescFromComment(comment); desc != "" {
					fieldSchema["description"] = desc
				}
			}
		}

		props[jsonName] = fieldSchema

		if !omitempty && !isPtr {
			required = append(required, jsonName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func parseJSONTag(tag string) (name string, omitempty bool) {
	jsonTag := reflect.StructTag(tag).Get("json")
	if jsonTag == "" {
		return "", false
	}
	parts := strings.Split(jsonTag, ",")
	name = parts[0]
	if name == "-" {
		return "-", false
	}
	for _, p := range parts[1:] {
		if strings.TrimSpace(p) == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty
}

// SnakeCase converts PascalCase to snake_case.
// Examples: "GetWeather" -> "get_weather", "ID" -> "id", "UserID" -> "user_id".
func SnakeCase(s string) string {
	if s == "" {
		return ""
	}
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					// Acronym boundary: e.g., "XML" in "XMLParser"
					result.WriteRune('_')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// extractEnumFromComment parses //tool:enum annotations from a comment string.
func extractEnumFromComment(comment string) []string {
	for _, line := range strings.Split(comment, "\n") {
		trimmed := strings.TrimSpace(line)
		stripped := strings.TrimPrefix(trimmed, "//")
		if strings.HasPrefix(stripped, "tool:enum") {
			parts := strings.SplitN(stripped, " ", 2)
			if len(parts) == 2 {
				vals := strings.Split(strings.TrimSpace(parts[1]), ",")
				result := make([]string, len(vals))
				for i, v := range vals {
					result[i] = strings.TrimSpace(v)
				}
				return result
			}
		}
	}
	return nil
}

// extractDescFromComment extracts description text from a comment string.
// It excludes lines containing //tool: directives.
func extractDescFromComment(comment string) string {
	var lines []string
	for _, line := range strings.Split(comment, "\n") {
		trimmed := strings.TrimSpace(line)
		stripped := strings.TrimPrefix(trimmed, "//")
		if strings.HasPrefix(stripped, "tool:") {
			continue
		}
		text := strings.TrimSpace(stripped)
		if text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// isIntegerKind checks if a basic kind is integer.
func isIntegerKind(kind types.BasicKind) bool {
	switch kind {
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr:
		return true
	}
	return false
}
