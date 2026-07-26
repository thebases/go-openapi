package core

type Example struct {
	Summary       string         `json:"summary,omitempty"`
	Description   string         `json:"description,omitempty"`
	Value         any            `json:"value,omitempty"`
	ExternalValue string         `json:"externalValue,omitempty"`
	Extensions    map[string]any `json:"-"`
}
