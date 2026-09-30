package core

// Document is the root OpenAPI Object. It marshals the fields defined by OAS 3.1/3.2; API.JSON downgrades the output for OAS 3.0.
type Document struct {
	OpenAPI           string                 `json:"openapi"`
	Info              Info                   `json:"info"`
	JSONSchemaDialect string                 `json:"jsonSchemaDialect,omitempty"`
	Servers           []Server               `json:"servers,omitempty"`
	Paths             map[string]*PathItem   `json:"paths,omitempty"`
	Webhooks          map[string]*PathItem   `json:"webhooks,omitempty"`
	Components        *Components            `json:"components,omitempty"`
	Security          []SecurityRequirement  `json:"security,omitempty"`
	Tags              []Tag                  `json:"tags,omitempty"`
	ExternalDocs      *ExternalDocumentation `json:"externalDocs,omitempty"`
	Extensions        map[string]any         `json:"-"`
}

// Info is the OpenAPI Info Object (API title, version, contact, and license).
type Info struct {
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	TermsOfService string         `json:"termsOfService,omitempty"`
	Contact        *Contact       `json:"contact,omitempty"`
	License        *License       `json:"license,omitempty"`
	Version        string         `json:"version"`
	Extensions     map[string]any `json:"-"`
}

// Contact is the OpenAPI Contact Object.
type Contact struct {
	Name       string         `json:"name,omitempty"`
	URL        string         `json:"url,omitempty"`
	Email      string         `json:"email,omitempty"`
	Extensions map[string]any `json:"-"`
}

// License is the OpenAPI License Object. Identifier (an SPDX expression) is OAS 3.1+ only.
type License struct {
	Name       string         `json:"name"`
	Identifier string         `json:"identifier,omitempty"`
	URL        string         `json:"url,omitempty"`
	Extensions map[string]any `json:"-"`
}

// Server is the OpenAPI Server Object.
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
	Extensions  map[string]any            `json:"-"`
}

// ServerVariable is the OpenAPI Server Variable Object.
type ServerVariable struct {
	Enum        []string       `json:"enum,omitempty"`
	Default     string         `json:"default"`
	Description string         `json:"description,omitempty"`
	Extensions  map[string]any `json:"-"`
}

// ExternalDocumentation is the OpenAPI External Documentation Object.
type ExternalDocumentation struct {
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url"`
	Extensions  map[string]any `json:"-"`
}

// Tag is the OpenAPI Tag Object.
type Tag struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Extensions   map[string]any         `json:"-"`
}
