package core

type PathItem struct {
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Get         *Operation             `json:"get,omitempty"`
	Put         *Operation             `json:"put,omitempty"`
	Post        *Operation             `json:"post,omitempty"`
	Delete      *Operation             `json:"delete,omitempty"`
	Options     *Operation             `json:"options,omitempty"`
	Head        *Operation             `json:"head,omitempty"`
	Patch       *Operation             `json:"patch,omitempty"`
	Trace       *Operation             `json:"trace,omitempty"`
	Servers     []Server               `json:"servers,omitempty"`
	Parameters  []ParameterOrReference `json:"parameters,omitempty"`
	Extensions  map[string]any         `json:"-"`
}

type Operation struct {
	Tags        []string                       `json:"tags,omitempty"`
	Summary     string                         `json:"summary,omitempty"`
	Description string                         `json:"description,omitempty"`
	OperationID string                         `json:"operationId,omitempty"`
	Parameters  []ParameterOrReference         `json:"parameters,omitempty"`
	RequestBody *RequestBodyOrReference        `json:"requestBody,omitempty"`
	Responses   map[string]ResponseOrReference `json:"responses"`
	Callbacks   map[string]any                 `json:"callbacks,omitempty"`
	Deprecated  bool                           `json:"deprecated,omitempty"`
	Security    []SecurityRequirement          `json:"security,omitempty"`
	Servers     []Server                       `json:"servers,omitempty"`
	Extensions  map[string]any                 `json:"-"`
}
