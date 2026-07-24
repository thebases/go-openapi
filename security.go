package openapi

type SecurityScheme struct {
	Type             string         `json:"type"`
	Description      string         `json:"description,omitempty"`
	Name             string         `json:"name,omitempty"`
	In               string         `json:"in,omitempty"`
	Scheme           string         `json:"scheme,omitempty"`
	BearerFormat     string         `json:"bearerFormat,omitempty"`
	Flows            *OAuthFlows    `json:"flows,omitempty"`
	OpenIDConnectURL string         `json:"openIdConnectUrl,omitempty"`
	Extensions       map[string]any `json:"-"`
}

type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type SecurityRequirement map[string][]string

func (a *API) AddBearerAuth(name, description string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.document.Components.SecuritySchemes == nil {
		a.document.Components.SecuritySchemes = make(map[string]*SecuritySchemeOrReference)
	}

	a.document.Components.SecuritySchemes[name] = &SecuritySchemeOrReference{
		Value: &SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  description,
		},
	}
}
