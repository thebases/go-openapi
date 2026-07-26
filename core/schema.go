package core

type Schema struct {
	Title            string   `json:"title,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMaximum *float64 `json:"exclusiveMaximum,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty"`
	ExclusiveMinimum *float64 `json:"exclusiveMinimum,omitempty"`

	MaxLength *uint64 `json:"maxLength,omitempty"`
	MinLength *uint64 `json:"minLength,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`

	MaxItems    *uint64 `json:"maxItems,omitempty"`
	MinItems    *uint64 `json:"minItems,omitempty"`
	UniqueItems bool    `json:"uniqueItems,omitempty"`

	MaxProperties *uint64 `json:"maxProperties,omitempty"`
	MinProperties *uint64 `json:"minProperties,omitempty"`

	Required []string `json:"required,omitempty"`
	Enum     []any    `json:"enum,omitempty"`
	Const    any      `json:"const,omitempty"`

	// Type follows JSON Schema Draft 2020-12: a single type name (string) or a
	// list of type names ([]string), which is how OAS 3.1 expresses nullability
	// (e.g. []string{"string", "null"}) instead of the removed 3.0 `nullable` keyword.
	Type   any    `json:"type,omitempty"`
	Format string `json:"format,omitempty"`

	AllOf []*SchemaOrReference `json:"allOf,omitempty"`
	OneOf []*SchemaOrReference `json:"oneOf,omitempty"`
	AnyOf []*SchemaOrReference `json:"anyOf,omitempty"`
	Not   *SchemaOrReference   `json:"not,omitempty"`

	Items                *SchemaOrReference            `json:"items,omitempty"`
	Properties           map[string]*SchemaOrReference `json:"properties,omitempty"`
	AdditionalProperties any                           `json:"additionalProperties,omitempty"`

	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
	WriteOnly   bool   `json:"writeOnly,omitempty"`
	// Example is deprecated by OAS 3.1 in favor of Examples; kept for compatibility.
	Example    any    `json:"example,omitempty"`
	Examples   []any  `json:"examples,omitempty"`
	Deprecated bool   `json:"deprecated,omitempty"`

	Discriminator *Discriminator         `json:"discriminator,omitempty"`
	XML           *XML                   `json:"xml,omitempty"`
	ExternalDocs  *ExternalDocumentation `json:"externalDocs,omitempty"`
	Extensions    map[string]any         `json:"-"`
}

type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

type XML struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty"`
}

// MakeNullable returns a JSON Schema 2020-12 type list that includes "null"
// alongside the given type name, matching how OAS 3.1 expresses nullability.
func MakeNullable(typeName string) []string {
	return []string{typeName, "null"}
}

// nullableType adds "null" to a schema's Type, whether it is currently unset,
// a single type name, or already a list of type names.
func nullableType(t any) any {
	switch value := t.(type) {
	case nil:
		return nil
	case string:
		if value == "null" {
			return value
		}
		return []string{value, "null"}
	case []string:
		for _, entry := range value {
			if entry == "null" {
				return value
			}
		}
		return append(append([]string{}, value...), "null")
	default:
		return t
	}
}

// primaryTypeName returns the first non-null type name from a Schema.Type
// value, for callers that need a single type hint (e.g. literal parsing).
func primaryTypeName(t any) string {
	switch value := t.(type) {
	case string:
		return value
	case []string:
		for _, entry := range value {
			if entry != "null" {
				return entry
			}
		}
	}
	return ""
}
