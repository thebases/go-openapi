package openapi

type Components struct {
	Schemas         map[string]*SchemaOrReference         `json:"schemas,omitempty"`
	Responses       map[string]*ResponseOrReference       `json:"responses,omitempty"`
	Parameters      map[string]*ParameterOrReference      `json:"parameters,omitempty"`
	Examples        map[string]*ExampleOrReference        `json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBodyOrReference    `json:"requestBodies,omitempty"`
	Headers         map[string]any                        `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecuritySchemeOrReference `json:"securitySchemes,omitempty"`
	Links           map[string]any                        `json:"links,omitempty"`
	Callbacks       map[string]any                        `json:"callbacks,omitempty"`
	Extensions      map[string]any                        `json:"-"`
}
