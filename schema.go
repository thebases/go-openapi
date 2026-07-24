package openapi

type Schema struct {
	Title            string   `json:"title,omitempty"`
	MultipleOf       *float64 `json:"multipleOf,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	ExclusiveMaximum bool     `json:"exclusiveMaximum,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty"`
	ExclusiveMinimum bool     `json:"exclusiveMinimum,omitempty"`

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

	Type   string `json:"type,omitempty"`
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
	Nullable    bool   `json:"nullable,omitempty"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
	WriteOnly   bool   `json:"writeOnly,omitempty"`
	Example     any    `json:"example,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`

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
