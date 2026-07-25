package core

type Parameter struct {
	Name            string                         `json:"name"`
	In              string                         `json:"in"`
	Description     string                         `json:"description,omitempty"`
	Required        bool                           `json:"required,omitempty"`
	Deprecated      bool                           `json:"deprecated,omitempty"`
	AllowEmptyValue bool                           `json:"allowEmptyValue,omitempty"`
	Style           string                         `json:"style,omitempty"`
	Explode         *bool                          `json:"explode,omitempty"`
	AllowReserved   bool                           `json:"allowReserved,omitempty"`
	Schema          *SchemaOrReference             `json:"schema,omitempty"`
	Example         any                            `json:"example,omitempty"`
	Examples        map[string]*ExampleOrReference `json:"examples,omitempty"`
	Extensions      map[string]any                 `json:"-"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
	Extensions  map[string]any       `json:"-"`
}

type MediaType struct {
	Schema     *SchemaOrReference             `json:"schema,omitempty"`
	Example    any                            `json:"example,omitempty"`
	Examples   map[string]*ExampleOrReference `json:"examples,omitempty"`
	Encoding   map[string]any                 `json:"encoding,omitempty"`
	Extensions map[string]any                 `json:"-"`
}

type Response struct {
	Description string               `json:"description"`
	Headers     map[string]any       `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Links       map[string]any       `json:"links,omitempty"`
	Extensions  map[string]any       `json:"-"`
}
