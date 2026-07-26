package main

import (
	"encoding/base64"
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	core "github.com/thebases/go-openapi/core"
	api "github.com/thebases/go-openapi/integrations/fiber"
)

// Fixed sample credentials for the docs "Try it" panel. These are demo-only
// constants, not real secrets, and exist so every Security Scheme Object case
// from the OpenAPI 3.2.0 spec can be exercised end-to-end against a running
// handler instead of only being declared in the document.
const (
	sampleAPIKey      = "sample-api-key-123"
	sampleBasicUser   = "sample-user"
	sampleBasicPass   = "sample-pass"
	sampleBearerToken = "sample-bearer-token-123"
	sampleOAuthToken  = "sample-oauth-token-456"
)

// checkBasicAuth decodes a "Basic <base64(user:pass)>" Authorization header
// and compares it against the fixed sample credentials.
func checkBasicAuth(authHeader string) bool {
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		return false
	}
	user, pass, ok := strings.Cut(string(decoded), ":")
	return ok && user == sampleBasicUser && pass == sampleBasicPass
}

// checkBearerAuth extracts the token from a "Bearer <token>" Authorization
// header and compares it against the expected fixed sample token.
func checkBearerAuth(authHeader, expected string) bool {
	const prefix = "Bearer "
	return strings.HasPrefix(authHeader, prefix) && authHeader[len(prefix):] == expected
}

func main() {
	app := fiber.New()
	doc := core.New(
		core.WithTitle("Fiber Example API"),
		core.WithDescription("descriptions/api.md"),
		core.WithVersion("1.0.0"),
		core.WithDocStyle(core.DocsBase),
		core.WithServer("http://localhost:3000", "Server"),
		core.WithServer("https://edc.thebase.vn", "Server"),
	)
	for name, scheme := range map[string]*core.SecuritySchemeOrReference{
		// The sample docs UI stores this credential in session storage as "apikey"
		// and applies it to header/query apiKey schemes for secured operations.
		// Fixed sample value: sample-api-key-123.
		"merchantApiKey": {
			Value: &core.SecurityScheme{
				Type:        "apiKey",
				Name:        "X-API-Key",
				In:          "header",
				Description: "Sample API key passed as the X-API-Key header. Fixed sample value: sample-api-key-123.",
			},
		},
		// HTTP Basic auth case from the Security Scheme Object spec.
		// Fixed sample credentials: sample-user / sample-pass.
		"merchantBasicAuth": {
			Value: &core.SecurityScheme{
				Type:        "http",
				Scheme:      "basic",
				Description: "Sample HTTP Basic auth. Fixed sample credentials: sample-user / sample-pass.",
			},
		},
		// HTTP Bearer auth case, distinct from the oauth2 scheme below.
		// Fixed sample token: sample-bearer-token-123.
		"merchantBearerAuth": {
			Value: &core.SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Sample HTTP Bearer auth. Fixed sample token: sample-bearer-token-123.",
			},
		},
		// mutualTLS case: declared for spec coverage; there is no fixed sample
		// credential to try since it requires a real client certificate.
		"merchantMutualTLS": {
			Value: &core.SecurityScheme{
				Type:        "mutualTLS",
				Description: "Sample mutual TLS scheme. Cert must be signed by example.com CA (docs-only, not enforced).",
			},
		},
		// openIdConnect case: declared for spec coverage; discovery/token
		// exchange against a real IdP is out of scope for this sample.
		"merchantOpenIdConnect": {
			Value: &core.SecurityScheme{
				Type:             "openIdConnect",
				OpenIDConnectURL: "https://auth.example.com/.well-known/openid-configuration",
				Description:      "Sample OpenID Connect discovery scheme (docs-only, not enforced).",
			},
		},
		// The sample docs UI stores this credential in session storage as "oauthkey"
		// and sends it as Bearer auth for oauth2 secured operations. All five
		// OAuth Flow Object cases from the spec are populated on one scheme.
		// Fixed sample access token: sample-oauth-token-456.
		"merchantOAuth": {
			Value: &core.SecurityScheme{
				Type:        "oauth2",
				Description: "Sample OAuth2 flows for merchant operations. Fixed sample access token: sample-oauth-token-456.",
				Flows: &core.OAuthFlows{
					Implicit: &core.OAuthFlow{
						AuthorizationURL: "https://auth.example.com/oauth/authorize",
						Scopes: map[string]string{
							"merchant:read":  "Read merchant data",
							"merchant:write": "Create and update merchants",
						},
					},
					Password: &core.OAuthFlow{
						TokenURL: "https://auth.example.com/oauth/token",
						Scopes: map[string]string{
							"merchant:read":  "Read merchant data",
							"merchant:write": "Create and update merchants",
						},
					},
					ClientCredentials: &core.OAuthFlow{
						TokenURL: "https://auth.example.com/oauth/token",
						Scopes: map[string]string{
							"merchant:read":  "Read merchant data",
							"merchant:write": "Create and update merchants",
						},
					},
					AuthorizationCode: &core.OAuthFlow{
						AuthorizationURL: "https://auth.example.com/oauth/authorize",
						TokenURL:         "https://auth.example.com/oauth/token",
						Scopes: map[string]string{
							"merchant:read":  "Read merchant data",
							"merchant:write": "Create and update merchants",
						},
					},
					DeviceAuthorization: &core.OAuthFlow{
						DeviceAuthorizationURL: "https://auth.example.com/oauth/device/authorize",
						TokenURL:               "https://auth.example.com/oauth/token",
						Scopes: map[string]string{
							"merchant:read":  "Read merchant data",
							"merchant:write": "Create and update merchants",
						},
					},
				},
			},
		},
	} {
		if err := doc.RegisterSecurityScheme(name, scheme); err != nil {
			log.Fatal(err)
		}
	}
	if err := doc.RegisterSchema("Merchant", core.InlineSchema(&core.Schema{
		Type:        "object",
		Description: "descriptions/merchant-schema.md",
		Example: map[string]any{
			"id":     "mrc_schema_001",
			"name":   "Schema Merchant",
			"status": "active",
		},
		Properties: map[string]*core.SchemaOrReference{
			"id":     core.InlineSchema(&core.Schema{Type: "string", Example: "mrc_schema_001"}),
			"name":   core.InlineSchema(&core.Schema{Type: "string", Example: "Schema Merchant"}),
			"status": core.InlineSchema(&core.Schema{Type: "string", Enum: []any{"active", "inactive"}, Example: "active"}),
		},
		Required: []string{"id", "name", "status"},
	})); err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterSchema("MerchantUpsertRequest", core.InlineSchema(&core.Schema{
		Type: "object",
		Example: map[string]any{
			"name":   "The Base Labs",
			"status": "active",
		},
		Properties: map[string]*core.SchemaOrReference{
			"name":   core.InlineSchema(&core.Schema{Type: "string", Example: "The Base Labs"}),
			"status": core.InlineSchema(&core.Schema{Type: "string", Enum: []any{"active", "inactive"}, Example: "active"}),
		},
		Required: []string{"name", "status"},
	})); err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterSchema("AuthSession", core.InlineSchema(&core.Schema{
		Type: "object",
		Example: map[string]any{
			"subject":   "svc_docs_demo",
			"authType":  "apiKey",
			"apiKeyId":  "key_live_docs",
			"scopes":    []string{"merchant:read"},
			"issuedFor": "docs-example",
		},
		Properties: map[string]*core.SchemaOrReference{
			"subject":   core.InlineSchema(&core.Schema{Type: "string", Example: "svc_docs_demo"}),
			"authType":  core.InlineSchema(&core.Schema{Type: "string", Enum: []any{"apiKey", "oauth2"}, Example: "apiKey"}),
			"apiKeyId":  core.InlineSchema(&core.Schema{Type: "string", Example: "key_live_docs"}),
			"tokenType": core.InlineSchema(&core.Schema{Type: "string", Example: "Bearer"}),
			"scopes":    core.ArraySchema(core.InlineSchema(&core.Schema{Type: "string", Example: "merchant:read"})),
			"issuedFor": core.InlineSchema(&core.Schema{Type: "string", Example: "docs-example"}),
		},
		Required: []string{"subject", "authType", "scopes", "issuedFor"},
	})); err != nil {
		log.Fatal(err)
	}
	// Reusable component examples keep the sample routes compact while exercising
	// the OAS 3.1.1 precedence rules for referenced media-type examples.
	for name, example := range map[string]*core.ExampleOrReference{
		"MerchantLive": core.InlineExample(&core.Example{
			Summary: "Live merchant",
			Value: map[string]any{
				"id":     "mrc_live_001",
				"name":   "The Base",
				"status": "active",
			},
		}),
		"MerchantSandbox": core.InlineExample(&core.Example{
			Summary: "Sandbox merchant",
			Value: map[string]any{
				"id":     "mrc_sandbox_001",
				"name":   "Sandbox Merchant",
				"status": "inactive",
			},
		}),
		"MerchantCreate": core.InlineExample(&core.Example{
			Summary: "Create merchant request",
			Value: map[string]any{
				"name":   "The Base Labs",
				"status": "active",
			},
		}),
		"ApiKeySession": core.InlineExample(&core.Example{
			Summary: "Authenticated API key session",
			Value: map[string]any{
				"subject":   "svc_docs_demo",
				"authType":  "apiKey",
				"apiKeyId":  "key_live_docs",
				"scopes":    []string{"merchant:read"},
				"issuedFor": "docs-example",
			},
		}),
		"OAuthSession": core.InlineExample(&core.Example{
			Summary: "Authenticated OAuth session",
			Value: map[string]any{
				"subject":   "user_123",
				"authType":  "oauth2",
				"tokenType": "Bearer",
				"scopes":    []string{"merchant:read", "merchant:write"},
				"issuedFor": "docs-example",
			},
		}),
	} {
		if err := doc.RegisterExample(name, example); err != nil {
			log.Fatal(err)
		}
	}
	err := api.GET(app, doc, core.Route("/merchants/:id", core.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Description: "descriptions/merchant-operation.md",
		Security:    []core.SecurityRequirement{{"merchantApiKey": {}}},
		Parameters: []core.ParameterOrReference{
			{Value: &core.Parameter{
				Name:        "id",
				In:          "path",
				Required:    true,
				Description: "descriptions/merchant-id.md",
				Schema:      core.InlineSchema(&core.Schema{Type: "string", Example: "mrc_schema_001"}),
				Example:     "mrc_live_001",
			}},
			{Value: &core.Parameter{
				Name:   "env",
				In:     "query",
				Schema: core.InlineSchema(&core.Schema{Type: "string", Enum: []any{"live", "sandbox"}, Example: "live"}),
				Examples: map[string]*core.ExampleOrReference{
					"live":    core.InlineExample(&core.Example{Summary: "Live data", Value: "live"}),
					"sandbox": core.InlineExample(&core.Example{Summary: "Sandbox data", Value: "sandbox"}),
				},
			}},
		},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "descriptions/merchant-response.md",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema: core.RefSchema("Merchant"),
							Examples: map[string]*core.ExampleOrReference{
								"live":    core.ExampleRef("MerchantLive"),
								"sandbox": core.ExampleRef("MerchantSandbox"),
							},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		name := "The Base"
		status := "active"
		if c.Query("env") == "sandbox" {
			name = "Sandbox Merchant"
			status = "inactive"
		}
		return c.JSON(fiber.Map{"id": c.Params("id"), "name": name, "status": status})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(app, doc, core.Route("/auth/apikey/profile", core.Operation{
		OperationID: "getApiKeyProfile",
		Summary:     "Get API key profile",
		Description: "Dedicated sample endpoint that requires the configured apiKey credential.",
		Security:    []core.SecurityRequirement{{"merchantApiKey": {}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Resolved API key session",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema: core.RefSchema("AuthSession"),
							Examples: map[string]*core.ExampleOrReference{
								"default": core.ExampleRef("ApiKeySession"),
							},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		// Enforces the fixed sample apiKey so the docs "Try it" panel round-trips.
		if c.Get("X-API-Key") != sampleAPIKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing X-API-Key"})
		}
		return c.JSON(fiber.Map{
			"subject":   "svc_docs_demo",
			"authType":  "apiKey",
			"apiKeyId":  "key_live_docs",
			"scopes":    []string{"merchant:read"},
			"issuedFor": "docs-example",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(app, doc, core.Route("/auth/basic/profile", core.Operation{
		OperationID: "getBasicProfile",
		Summary:     "Get basic auth profile",
		Description: "Dedicated sample endpoint that requires the configured HTTP Basic credential.",
		Security:    []core.SecurityRequirement{{"merchantBasicAuth": {}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Resolved basic-auth session",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema:  core.RefSchema("AuthSession"),
							Example: map[string]any{"subject": "sample-user", "authType": "basic", "scopes": []string{}, "issuedFor": "docs-example"},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		if !checkBasicAuth(c.Get("Authorization")) {
			c.Set("WWW-Authenticate", `Basic realm="docs-example"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing basic auth credentials"})
		}
		return c.JSON(fiber.Map{
			"subject":   sampleBasicUser,
			"authType":  "basic",
			"scopes":    []string{},
			"issuedFor": "docs-example",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(app, doc, core.Route("/auth/bearer/profile", core.Operation{
		OperationID: "getBearerProfile",
		Summary:     "Get bearer auth profile",
		Description: "Dedicated sample endpoint that requires the configured HTTP Bearer credential (distinct from the oauth2 scheme).",
		Security:    []core.SecurityRequirement{{"merchantBearerAuth": {}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Resolved bearer-auth session",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema:  core.RefSchema("AuthSession"),
							Example: map[string]any{"subject": "svc_docs_demo", "authType": "bearer", "tokenType": "Bearer", "scopes": []string{}, "issuedFor": "docs-example"},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		if !checkBearerAuth(c.Get("Authorization"), sampleBearerToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing bearer token"})
		}
		return c.JSON(fiber.Map{
			"subject":   "svc_docs_demo",
			"authType":  "bearer",
			"tokenType": "Bearer",
			"scopes":    []string{},
			"issuedFor": "docs-example",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	// mutualTLS and openIdConnect are declared purely for Security Scheme Object
	// spec coverage: neither can be satisfied by a fixed sample key, so these
	// routes document the scheme without enforcing it.
	err = api.GET(app, doc, core.Route("/auth/mtls/profile", core.Operation{
		OperationID: "getMutualTLSProfile",
		Summary:     "Get mutualTLS profile (docs-only)",
		Description: "Documents the mutualTLS Security Scheme Object case; not enforced by this sample.",
		Security:    []core.SecurityRequirement{{"merchantMutualTLS": {}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Static sample response",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema:  core.RefSchema("AuthSession"),
							Example: map[string]any{"subject": "docs-only", "authType": "mutualTLS", "scopes": []string{}, "issuedFor": "docs-example"},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"subject": "docs-only", "authType": "mutualTLS", "scopes": []string{}, "issuedFor": "docs-example"})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(app, doc, core.Route("/auth/oidc/profile", core.Operation{
		OperationID: "getOpenIDConnectProfile",
		Summary:     "Get openIdConnect profile (docs-only)",
		Description: "Documents the openIdConnect Security Scheme Object case; not enforced by this sample.",
		Security:    []core.SecurityRequirement{{"merchantOpenIdConnect": {}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Static sample response",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema:  core.RefSchema("AuthSession"),
							Example: map[string]any{"subject": "docs-only", "authType": "openIdConnect", "scopes": []string{}, "issuedFor": "docs-example"},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"subject": "docs-only", "authType": "openIdConnect", "scopes": []string{}, "issuedFor": "docs-example"})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(app, doc, core.Route("/auth/oauth/session", core.Operation{
		OperationID: "getOAuthSession",
		Summary:     "Get OAuth session",
		Description: "Dedicated sample endpoint that requires the configured oauth2 credential.",
		Security:    []core.SecurityRequirement{{"merchantOAuth": {"merchant:read"}}},
		Responses: map[string]core.ResponseOrReference{
			"200": {
				Value: &core.Response{
					Description: "Resolved OAuth session",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema: core.RefSchema("AuthSession"),
							Examples: map[string]*core.ExampleOrReference{
								"default": core.ExampleRef("OAuthSession"),
							},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		if !checkBearerAuth(c.Get("Authorization"), sampleOAuthToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or missing oauth2 bearer token"})
		}
		return c.JSON(fiber.Map{
			"subject":   "user_123",
			"authType":  "oauth2",
			"tokenType": "Bearer",
			"scopes":    []string{"merchant:read", "merchant:write"},
			"issuedFor": "docs-example",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.POST(app, doc, core.Route("/merchants", core.Operation{
		OperationID: "createMerchant",
		Summary:     "Create merchant",
		Description: "Creates a merchant and demonstrates requestBody examples in OpenAPI 3.1.1.",
		Security:    []core.SecurityRequirement{{"merchantOAuth": {"merchant:write"}}},
		RequestBody: &core.RequestBodyOrReference{
			Value: &core.RequestBody{
				Description: "Create merchant payload.",
				Required:    true,
				Content: map[string]core.MediaType{
					"application/json": {
						Schema: core.RefSchema("MerchantUpsertRequest"),
						Examples: map[string]*core.ExampleOrReference{
							"standard": core.ExampleRef("MerchantCreate"),
							"minimal": core.InlineExample(&core.Example{
								Summary: "Minimal request",
								Value: map[string]any{
									"name":   "Pilot Merchant",
									"status": "inactive",
								},
							}),
						},
					},
				},
			},
		},
		Responses: map[string]core.ResponseOrReference{
			"201": {
				Value: &core.Response{
					Description: "Merchant created",
					Content: map[string]core.MediaType{
						"application/json": {
							Schema:  core.RefSchema("Merchant"),
							Example: map[string]any{"id": "mrc_created_001", "name": "The Base Labs", "status": "active"},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
		return c.Status(201).JSON(fiber.Map{
			"id":     "mrc_created_001",
			"name":   "The Base Labs",
			"status": "active",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(app.Listen(":3000"))
}
