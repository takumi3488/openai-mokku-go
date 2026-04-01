package main

import (
	"testing"
)

func TestGenerateDummyValue_StringType_ReturnsString(t *testing.T) {
	// Given
	schema := map[string]interface{}{"type": "string"}
	// When
	got := generateDummyValue(schema)
	// Then
	if _, ok := got.(string); !ok {
		t.Errorf("expected string, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_NumberType_ReturnsFloat(t *testing.T) {
	// Given
	schema := map[string]interface{}{"type": "number"}
	// When
	got := generateDummyValue(schema)
	// Then
	if _, ok := got.(float64); !ok {
		t.Errorf("expected float64, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_IntegerType_ReturnsIntLike(t *testing.T) {
	// Given
	schema := map[string]interface{}{"type": "integer"}
	// When
	got := generateDummyValue(schema)
	// Then: int, int64, or float64 (JSON number) are all acceptable
	switch got.(type) {
	case int, int64, float64:
		// ok
	default:
		t.Errorf("expected integer-like type, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_BooleanType_ReturnsBool(t *testing.T) {
	// Given
	schema := map[string]interface{}{"type": "boolean"}
	// When
	got := generateDummyValue(schema)
	// Then
	if _, ok := got.(bool); !ok {
		t.Errorf("expected bool, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_ArrayType_ReturnsSlice(t *testing.T) {
	// Given
	schema := map[string]interface{}{
		"type":  "array",
		"items": map[string]interface{}{"type": "string"},
	}
	// When
	got := generateDummyValue(schema)
	// Then
	if _, ok := got.([]interface{}); !ok {
		t.Errorf("expected []interface{}, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_ObjectType_ReturnsMap(t *testing.T) {
	// Given
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "integer"},
		},
		"required": []interface{}{"name", "age"},
	}
	// When
	got := generateDummyValue(schema)
	// Then: result is a map
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T: %v", got, got)
	}
	// Required fields must be present
	if _, ok := m["name"]; !ok {
		t.Error("expected 'name' field in result")
	}
	if _, ok := m["age"]; !ok {
		t.Error("expected 'age' field in result")
	}
}

func TestGenerateDummyValue_EnumType_ReturnsValidEnumValue(t *testing.T) {
	// Given: schema with enum values
	schema := map[string]interface{}{
		"type": "string",
		"enum": []interface{}{"apple", "banana", "cherry"},
	}
	// When
	got := generateDummyValue(schema)
	// Then: returns one of the enum values
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string, got %T", got)
	}
	validValues := map[string]bool{"apple": true, "banana": true, "cherry": true}
	if !validValues[s] {
		t.Errorf("expected one of enum values, got %q", s)
	}
}

func TestGenerateDummyValue_NullableString_ReturnsString(t *testing.T) {
	// Given: nullable string (the mock should return a non-nil string)
	schema := map[string]interface{}{
		"type":     "string",
		"nullable": true,
	}
	// When
	got := generateDummyValue(schema)
	// Then: returns a string, not nil
	if _, ok := got.(string); !ok {
		t.Errorf("expected string for nullable string, got %T: %v", got, got)
	}
}

func TestGenerateDummyValue_UnknownType_ReturnsNonNilPlaceholder(t *testing.T) {
	// Given: unknown/unsupported type - should not panic
	schema := map[string]interface{}{"type": "unknown_future_type"}
	// When
	got := generateDummyValue(schema)
	// Then: returns some non-nil placeholder
	if got == nil {
		t.Error("expected non-nil placeholder for unknown type")
	}
}

func TestGenerateDummyValue_NestedObject_PopulatesAllLevels(t *testing.T) {
	// Given: nested object schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"name"},
			},
		},
		"required": []interface{}{"user"},
	}
	// When
	got := generateDummyValue(schema)
	// Then: nested structure is populated
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	user, ok := m["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'user' to be a map, got %T", m["user"])
	}
	if _, ok := user["name"]; !ok {
		t.Error("expected 'name' in nested 'user' object")
	}
}

func TestGenerateDummyValue_IsDeterministic(t *testing.T) {
	// Given: same schema
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"value": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"value"},
	}
	// When: called twice
	first := generateDummyValue(schema)
	second := generateDummyValue(schema)
	// Then: same result (map[string]interface{} compared by string field)
	m1, ok1 := first.(map[string]interface{})
	m2, ok2 := second.(map[string]interface{})
	if !ok1 || !ok2 {
		t.Fatalf("expected maps, got %T and %T", first, second)
	}
	v1, ok1 := m1["value"].(string)
	v2, ok2 := m2["value"].(string)
	if !ok1 || !ok2 {
		t.Fatalf("expected string values, got %T and %T", m1["value"], m2["value"])
	}
	if v1 != v2 {
		t.Errorf("expected deterministic output: %q vs %q", v1, v2)
	}
}
