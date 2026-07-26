package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	core "github.com/thebases/go-openapi/core"
	api "github.com/thebases/go-openapi/integrations/chi"
)

func main() {
	router := chi.NewRouter()
	doc := core.New(
		core.WithTitle("Chi Example API"),
		core.WithDescription("descriptions/api.md"),
		core.WithVersion("1.0.0"),
		core.WithDocStyle(core.DocsSwagger),
	)
	for name, scheme := range map[string]*core.SecuritySchemeOrReference{
		// The sample docs UI stores this credential in session storage as "apikey"
		// and applies it to header/query apiKey schemes for secured operations.
		"merchantApiKey": {
			Value: &core.SecurityScheme{
				Type:        "apiKey",
				Name:        "X-API-Key",
				In:          "header",
				Description: "Sample API key passed as the X-API-Key header.",
			},
		},
		// The sample docs UI stores this credential in session storage as "oauthkey"
		// and sends it as Bearer auth for oauth2 secured operations.
		"merchantOAuth": {
			Value: &core.SecurityScheme{
				Type:        "oauth2",
				Description: "Sample OAuth2 authorization-code flow for merchant write operations.",
				Flows: &core.OAuthFlows{
					AuthorizationCode: &core.OAuthFlow{
						AuthorizationURL: "https://auth.example.com/oauth/authorize",
						TokenURL:         "https://auth.example.com/oauth/token",
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
	// the OAS 3.0.3 precedence rules for referenced media-type examples.
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
	err := api.GET(router, doc, core.Route("/merchants/{id}", core.Operation{
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
	}), func(w http.ResponseWriter, r *http.Request) {
		name := "The Base"
		status := "active"
		if r.URL.Query().Get("env") == "sandbox" {
			name = "Sandbox Merchant"
			status = "inactive"
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":     chi.URLParam(r, "id"),
			"name":   name,
			"status": status,
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	err = api.GET(router, doc, core.Route("/auth/apikey/profile", core.Operation{
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
	}), func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
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

	err = api.GET(router, doc, core.Route("/auth/oauth/session", core.Operation{
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
	}), func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
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

	err = api.POST(router, doc, core.Route("/merchants", core.Operation{
		OperationID: "createMerchant",
		Summary:     "Create merchant",
		Description: "Creates a merchant and demonstrates requestBody examples in OpenAPI 3.0.3.",
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
	}), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":     "mrc_created_001",
			"name":   "The Base Labs",
			"status": "active",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(":3000", router))
}
