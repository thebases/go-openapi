package core

// SecurityScheme is the OpenAPI Security Scheme Object.
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

// OAuthFlows is the OpenAPI OAuth Flows Object. DeviceAuthorization is OAS 3.2 only.
type OAuthFlows struct {
	Implicit            *OAuthFlow `json:"implicit,omitempty"`
	Password            *OAuthFlow `json:"password,omitempty"`
	ClientCredentials   *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode   *OAuthFlow `json:"authorizationCode,omitempty"`
	DeviceAuthorization *OAuthFlow `json:"deviceAuthorization,omitempty"`
}

// OAuthFlow is the OpenAPI OAuth Flow Object.
type OAuthFlow struct {
	AuthorizationURL       string            `json:"authorizationUrl,omitempty"`
	DeviceAuthorizationURL string            `json:"deviceAuthorizationUrl,omitempty"`
	TokenURL               string            `json:"tokenUrl,omitempty"`
	RefreshURL             string            `json:"refreshUrl,omitempty"`
	Scopes                 map[string]string `json:"scopes"`
}

// SecurityRequirement is the OpenAPI Security Requirement Object: scheme name to required scopes.
type SecurityRequirement map[string][]string
