package core

type Document struct {
	OpenAPI      string                 `json:"openapi"`
	Info         Info                   `json:"info"`
	Servers      []Server               `json:"servers,omitempty"`
	Paths        map[string]*PathItem   `json:"paths"`
	Components   *Components            `json:"components,omitempty"`
	Security     []SecurityRequirement  `json:"security,omitempty"`
	Tags         []Tag                  `json:"tags,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Extensions   map[string]any         `json:"-"`
}

type Info struct {
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	TermsOfService string         `json:"termsOfService,omitempty"`
	Contact        *Contact       `json:"contact,omitempty"`
	License        *License       `json:"license,omitempty"`
	Version        string         `json:"version"`
	Extensions     map[string]any `json:"-"`
}

type Contact struct {
	Name       string         `json:"name,omitempty"`
	URL        string         `json:"url,omitempty"`
	Email      string         `json:"email,omitempty"`
	Extensions map[string]any `json:"-"`
}

type License struct {
	Name       string         `json:"name"`
	URL        string         `json:"url,omitempty"`
	Extensions map[string]any `json:"-"`
}

type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
	Extensions  map[string]any            `json:"-"`
}

type ServerVariable struct {
	Enum        []string       `json:"enum,omitempty"`
	Default     string         `json:"default"`
	Description string         `json:"description,omitempty"`
	Extensions  map[string]any `json:"-"`
}

type ExternalDocumentation struct {
	Description string         `json:"description,omitempty"`
	URL         string         `json:"url"`
	Extensions  map[string]any `json:"-"`
}

type Tag struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	Extensions   map[string]any         `json:"-"`
}
