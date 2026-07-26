package core

import "encoding/json"

// downgradeToVersion30 rewrites a marshaled OAS 3.1/3.2-shaped document into
// the OAS 3.0 Schema Object subset. It walks the generic JSON tree (rather
// than a parallel Go type) so every schema location — components, parameters,
// requestBody, responses, callbacks — is covered by one recursive pass driven
// purely by JSON key names.
func downgradeToVersion30(raw []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	// jsonSchemaDialect, webhooks, components.pathItems, and license.identifier
	// are 3.1+/3.2-only fields with no OAS 3.0 equivalent.
	delete(doc, "jsonSchemaDialect")
	delete(doc, "webhooks")
	if info, ok := doc["info"].(map[string]any); ok {
		if license, ok := info["license"].(map[string]any); ok {
			delete(license, "identifier")
		}
	}
	if components, ok := doc["components"].(map[string]any); ok {
		delete(components, "pathItems")
	}

	downgradeValue("", doc)
	return json.MarshalIndent(doc, "", "  ")
}

// downgradeValue recurses through the document using JSON key names to find
// schema-bearing fields ("schema", "schemas", "properties", "items", "not",
// "additionalProperties", "allOf"/"oneOf"/"anyOf"), and downgrades each schema
// node it finds. Non-schema fields (including SecurityScheme.type, which is
// also a bare string) are left untouched because they never carry these keys.
func downgradeValue(key string, value any) any {
	switch v := value.(type) {
	case map[string]any:
		switch key {
		case "schema", "items", "not", "additionalProperties":
			return downgradeSchema(v)
		case "schemas", "properties":
			for name, sub := range v {
				if schema, ok := sub.(map[string]any); ok {
					v[name] = downgradeSchema(schema)
				}
			}
			return v
		default:
			for childKey, sub := range v {
				v[childKey] = downgradeValue(childKey, sub)
			}
			return v
		}
	case []any:
		if key == "allOf" || key == "oneOf" || key == "anyOf" {
			for i, sub := range v {
				if schema, ok := sub.(map[string]any); ok {
					v[i] = downgradeSchema(schema)
				}
			}
			return v
		}
		for i, sub := range v {
			v[i] = downgradeValue("", sub)
		}
		return v
	default:
		return value
	}
}

// downgradeSchema rewrites one Schema Object from JSON Schema 2020-12 shape
// (type array with "null", numeric exclusiveMinimum/Maximum, examples list,
// const) into the OAS 3.0 Schema Object shape (nullable, boolean
// exclusiveMinimum/Maximum paired with minimum/maximum, single example), then
// recurses into its own nested schema fields.
func downgradeSchema(m map[string]any) map[string]any {
	// The reflector emits nullable $ref fields as oneOf:[{$ref}, {type:null}]
	// because 3.1 sibling keywords next to $ref are unreliable; 3.0 instead
	// pairs allOf:[{$ref}] with nullable:true.
	if oneOf, ok := m["oneOf"].([]any); ok && len(oneOf) == 2 && isNullSchemaNode(oneOf[1]) {
		if ref, ok := oneOf[0].(map[string]any); ok {
			delete(m, "oneOf")
			m["nullable"] = true
			m["allOf"] = []any{ref}
		}
	}

	if list, ok := m["type"].([]any); ok {
		var kept string
		nullable := false
		for _, entry := range list {
			name, _ := entry.(string)
			if name == "null" {
				nullable = true
			} else if kept == "" {
				kept = name
			}
		}
		if nullable {
			m["nullable"] = true
		}
		if kept != "" {
			m["type"] = kept
		} else {
			delete(m, "type")
		}
	}

	if num, ok := m["exclusiveMinimum"].(float64); ok {
		m["minimum"] = num
		m["exclusiveMinimum"] = true
	}
	if num, ok := m["exclusiveMaximum"].(float64); ok {
		m["maximum"] = num
		m["exclusiveMaximum"] = true
	}

	if examples, ok := m["examples"].([]any); ok {
		if _, hasExample := m["example"]; !hasExample && len(examples) > 0 {
			m["example"] = examples[0]
		}
		delete(m, "examples")
	}

	delete(m, "const") // no OAS 3.0 equivalent; best-effort drop

	for childKey, sub := range m {
		m[childKey] = downgradeValue(childKey, sub)
	}
	return m
}

func isNullSchemaNode(v any) bool {
	m, ok := v.(map[string]any)
	return ok && len(m) == 1 && m["type"] == "null"
}
