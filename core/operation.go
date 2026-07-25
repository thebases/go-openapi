package openapi

type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Trace   *Operation `json:"trace,omitempty"`
}

type Operation struct {
	Tags        []string       `json:"tags,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	OperationID string         `json:"operationId,omitempty"`
	Parameters  []Parameter    `json:"parameters,omitempty"`
	RequestBody *RequestBody   `json:"requestBody,omitempty"`
	Responses   Responses      `json:"responses"`
	Deprecated  bool           `json:"deprecated,omitempty"`
	Extensions  map[string]any `json:"-"`
}

type Responses map[string]Response
