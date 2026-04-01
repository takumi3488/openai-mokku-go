package main

import "encoding/json"

// generateDummyValue generates a deterministic dummy value that conforms to the given JSON Schema.
// Supported keywords: type, properties, required, items, enum.
// Unsupported keywords are ignored and a stable placeholder is returned.
func generateDummyValue(schema map[string]interface{}) interface{} {
	if schema == nil {
		return "placeholder"
	}

	schemaType, hasType := schema["type"]
	if hasType {
		switch t := schemaType.(type) {
		case string:
			switch t {
			case "string":
				if enumVals, ok := schema["enum"].([]interface{}); ok && len(enumVals) > 0 {
					// Return first enum value deterministically
					if s, ok := enumVals[0].(string); ok {
						return s
					}
					return enumVals[0]
				}
				return "dummy_string"
			case "number", "float", "double":
				return 1.5
			case "integer", "int":
				if enumVals, ok := schema["enum"].([]interface{}); ok && len(enumVals) > 0 {
					if n, ok := enumVals[0].(float64); ok {
						return n
					}
					return enumVals[0]
				}
				return 42
			case "boolean":
				return true
			case "array":
				if items, ok := schema["items"].(map[string]interface{}); ok {
					return []interface{}{generateDummyValue(items)}
				}
				return []interface{}{}
			case "object":
				return generateDummyObject(schema)
			case "null":
				return nil
			}
		}
	}

	// Fallback
	return "placeholder"
}

// generateDummyObject generates a deterministic dummy object that conforms to the given JSON Schema.
func generateDummyObject(schema map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return result
	}

	required, _ := schema["required"].([]interface{})
	requiredSet := make(map[string]bool)
	for _, r := range required {
		if rStr, ok := r.(string); ok {
			requiredSet[rStr] = true
		}
	}

	// Generate values for all properties that are required or just all properties
	for name, propSchema := range properties {
		if _, isRequired := requiredSet[name]; isRequired || len(required) == 0 {
			if prop, ok := propSchema.(map[string]interface{}); ok {
				result[name] = generateDummyValue(prop)
			}
		}
	}

	return result
}

// generateJSONSchemaResponse generates a JSON response conforming to the given JSON Schema.
func generateJSONSchemaResponse(schema map[string]interface{}) string {
	obj := generateDummyObject(schema)
	b, err := json.Marshal(obj)
	if err != nil {
		return `{}`
	}
	return string(b)
}

// generateJSONFromSchemaBytes unmarshals raw schema bytes and generates a conforming JSON string.
// Returns "{}" if the bytes are empty or cannot be parsed.
func generateJSONFromSchemaBytes(schemaBytes []byte) string {
	if len(schemaBytes) == 0 {
		return `{}`
	}
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &schemaMap); err != nil {
		return `{}`
	}
	return generateJSONSchemaResponse(schemaMap)
}
