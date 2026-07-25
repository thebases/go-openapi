package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	api "github.com/thebases/go-openapi/integrations/fiber"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	app := fiber.New()
	doc := openapi.New(
		openapi.WithTitle("Fiber Example API"),
		openapi.WithDescription("descriptions/api.md"),
		openapi.WithVersion("1.0.0"),
		openapi.WithDocStyle(openapi.DocsBase),
		openapi.WithServer("http://localhost:3000", "Server"),
		openapi.WithServer("https://edc.thebase.vn", "Server"),
	)
	for name, scheme := range map[string]*openapi.SecuritySchemeOrReference{
		// The sample docs UI stores this credential in session storage as "apikey"
		// and applies it to header/query apiKey schemes for secured operations.
		"merchantApiKey": {
			Value: &openapi.SecurityScheme{
				Type:        "apiKey",
				Name:        "X-API-Key",
				In:          "header",
				Description: "Sample API key passed as the X-API-Key header.",
			},
		},
		// The sample docs UI stores this credential in session storage as "oauthkey"
		// and sends it as Bearer auth for oauth2 secured operations.
		"merchantOAuth": {
			Value: &openapi.SecurityScheme{
				Type:        "oauth2",
				Description: "Sample OAuth2 authorization-code flow for merchant write operations.",
				Flows: &openapi.OAuthFlows{
					AuthorizationCode: &openapi.OAuthFlow{
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
	if err := doc.RegisterSchema("Merchant", openapi.InlineSchema(&openapi.Schema{
		Type:        "object",
		Description: "descriptions/merchant-schema.md",
		Example: map[string]any{
			"id":     "mrc_schema_001",
			"name":   "Schema Merchant",
			"status": "active",
		},
		Properties: map[string]*openapi.SchemaOrReference{
			"id":     openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "mrc_schema_001"}),
			"name":   openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "Schema Merchant"}),
			"status": openapi.InlineSchema(&openapi.Schema{Type: "string", Enum: []any{"active", "inactive"}, Example: "active"}),
		},
		Required: []string{"id", "name", "status"},
	})); err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterSchema("MerchantUpsertRequest", openapi.InlineSchema(&openapi.Schema{
		Type: "object",
		Example: map[string]any{
			"name":   "The Base Labs",
			"status": "active",
		},
		Properties: map[string]*openapi.SchemaOrReference{
			"name":   openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "The Base Labs"}),
			"status": openapi.InlineSchema(&openapi.Schema{Type: "string", Enum: []any{"active", "inactive"}, Example: "active"}),
		},
		Required: []string{"name", "status"},
	})); err != nil {
		log.Fatal(err)
	}
	if err := doc.RegisterSchema("AuthSession", openapi.InlineSchema(&openapi.Schema{
		Type: "object",
		Example: map[string]any{
			"subject":   "svc_docs_demo",
			"authType":  "apiKey",
			"apiKeyId":  "key_live_docs",
			"scopes":    []string{"merchant:read"},
			"issuedFor": "docs-example",
		},
		Properties: map[string]*openapi.SchemaOrReference{
			"subject":   openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "svc_docs_demo"}),
			"authType":  openapi.InlineSchema(&openapi.Schema{Type: "string", Enum: []any{"apiKey", "oauth2"}, Example: "apiKey"}),
			"apiKeyId":  openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "key_live_docs"}),
			"tokenType": openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "Bearer"}),
			"scopes":    openapi.ArraySchema(openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "merchant:read"})),
			"issuedFor": openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "docs-example"}),
		},
		Required: []string{"subject", "authType", "scopes", "issuedFor"},
	})); err != nil {
		log.Fatal(err)
	}
	// Reusable component examples keep the sample routes compact while exercising
	// the OAS 3.0.3 precedence rules for referenced media-type examples.
	for name, example := range map[string]*openapi.ExampleOrReference{
		"MerchantLive": openapi.InlineExample(&openapi.Example{
			Summary: "Live merchant",
			Value: map[string]any{
				"id":     "mrc_live_001",
				"name":   "The Base",
				"status": "active",
			},
		}),
		"MerchantSandbox": openapi.InlineExample(&openapi.Example{
			Summary: "Sandbox merchant",
			Value: map[string]any{
				"id":     "mrc_sandbox_001",
				"name":   "Sandbox Merchant",
				"status": "inactive",
			},
		}),
		"MerchantCreate": openapi.InlineExample(&openapi.Example{
			Summary: "Create merchant request",
			Value: map[string]any{
				"name":   "The Base Labs",
				"status": "active",
			},
		}),
		"ApiKeySession": openapi.InlineExample(&openapi.Example{
			Summary: "Authenticated API key session",
			Value: map[string]any{
				"subject":   "svc_docs_demo",
				"authType":  "apiKey",
				"apiKeyId":  "key_live_docs",
				"scopes":    []string{"merchant:read"},
				"issuedFor": "docs-example",
			},
		}),
		"OAuthSession": openapi.InlineExample(&openapi.Example{
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
	err := api.GET(app, doc, openapi.Route("/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Description: "descriptions/merchant-operation.md",
		Security:    []openapi.SecurityRequirement{{"merchantApiKey": {}}},
		Parameters: []openapi.ParameterOrReference{
			{Value: &openapi.Parameter{
				Name:        "id",
				In:          "path",
				Required:    true,
				Description: "descriptions/merchant-id.md",
				Schema:      openapi.InlineSchema(&openapi.Schema{Type: "string", Example: "mrc_schema_001"}),
				Example:     "mrc_live_001",
			}},
			{Value: &openapi.Parameter{
				Name:   "env",
				In:     "query",
				Schema: openapi.InlineSchema(&openapi.Schema{Type: "string", Enum: []any{"live", "sandbox"}, Example: "live"}),
				Examples: map[string]*openapi.ExampleOrReference{
					"live":    openapi.InlineExample(&openapi.Example{Summary: "Live data", Value: "live"}),
					"sandbox": openapi.InlineExample(&openapi.Example{Summary: "Sandbox data", Value: "sandbox"}),
				},
			}},
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": {
				Value: &openapi.Response{
					Description: "descriptions/merchant-response.md",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: openapi.RefSchema("Merchant"),
							Examples: map[string]*openapi.ExampleOrReference{
								"live":    openapi.ExampleRef("MerchantLive"),
								"sandbox": openapi.ExampleRef("MerchantSandbox"),
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

	err = api.GET(app, doc, openapi.Route("/auth/apikey/profile", openapi.Operation{
		OperationID: "getApiKeyProfile",
		Summary:     "Get API key profile",
		Description: "Dedicated sample endpoint that requires the configured apiKey credential.",
		Security:    []openapi.SecurityRequirement{{"merchantApiKey": {}}},
		Responses: map[string]openapi.ResponseOrReference{
			"200": {
				Value: &openapi.Response{
					Description: "Resolved API key session",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: openapi.RefSchema("AuthSession"),
							Examples: map[string]*openapi.ExampleOrReference{
								"default": openapi.ExampleRef("ApiKeySession"),
							},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
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

	err = api.GET(app, doc, openapi.Route("/auth/oauth/session", openapi.Operation{
		OperationID: "getOAuthSession",
		Summary:     "Get OAuth session",
		Description: "Dedicated sample endpoint that requires the configured oauth2 credential.",
		Security:    []openapi.SecurityRequirement{{"merchantOAuth": {"merchant:read"}}},
		Responses: map[string]openapi.ResponseOrReference{
			"200": {
				Value: &openapi.Response{
					Description: "Resolved OAuth session",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema: openapi.RefSchema("AuthSession"),
							Examples: map[string]*openapi.ExampleOrReference{
								"default": openapi.ExampleRef("OAuthSession"),
							},
						},
					},
				},
			},
		},
	}), func(c fiber.Ctx) error {
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

	err = api.POST(app, doc, openapi.Route("/merchants", openapi.Operation{
		OperationID: "createMerchant",
		Summary:     "Create merchant",
		Description: "Creates a merchant and demonstrates requestBody examples in OpenAPI 3.0.3.",
		Security:    []openapi.SecurityRequirement{{"merchantOAuth": {"merchant:write"}}},
		RequestBody: &openapi.RequestBodyOrReference{
			Value: &openapi.RequestBody{
				Description: "Create merchant payload.",
				Required:    true,
				Content: map[string]openapi.MediaType{
					"application/json": {
						Schema: openapi.RefSchema("MerchantUpsertRequest"),
						Examples: map[string]*openapi.ExampleOrReference{
							"standard": openapi.ExampleRef("MerchantCreate"),
							"minimal": openapi.InlineExample(&openapi.Example{
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
		Responses: map[string]openapi.ResponseOrReference{
			"201": {
				Value: &openapi.Response{
					Description: "Merchant created",
					Content: map[string]openapi.MediaType{
						"application/json": {
							Schema:  openapi.RefSchema("Merchant"),
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
