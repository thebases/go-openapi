package openapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestAPIAddOperation(t *testing.T) {
	api := New(
		WithTitle("Merchant API"),
		WithVersion("1.0.0"),
	)

	err := api.RegisterSchema("Merchant", InlineSchema(&Schema{
		Type: "object",
		Properties: map[string]*SchemaOrReference{
			"id":   StringSchema(),
			"name": StringSchema(),
		},
		Required: []string{"id", "name"},
	}))
	if err != nil {
		t.Fatalf("register schema: %v", err)
	}

	err = api.AddOperation(http.MethodGet, "/merchants/{id}", Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []ParameterOrReference{
			PathParameter("id", StringSchema()),
		},
		Responses: map[string]ResponseOrReference{
			"200": JSONResponse("Merchant returned", RefSchema("Merchant")),
		},
	})
	if err != nil {
		t.Fatalf("add operation: %v", err)
	}

	doc := api.Document()
	if doc.Info.Title != "Merchant API" {
		t.Fatalf("unexpected title: %q", doc.Info.Title)
	}
	if doc.Paths["/merchants/{id}"] == nil || doc.Paths["/merchants/{id}"].Get == nil {
		t.Fatalf("expected GET operation to be registered")
	}
	if doc.Components == nil || doc.Components.Schemas["Merchant"] == nil {
		t.Fatalf("expected Merchant schema to be registered")
	}
}

func TestAPIRegisterExample(t *testing.T) {
	api := New()

	err := api.RegisterExample("merchant-live", InlineExample(&Example{
		Summary: "Live merchant",
		Value: map[string]any{
			"id":   "mrc_live_001",
			"name": "The Base",
		},
	}))
	if err != nil {
		t.Fatalf("register example: %v", err)
	}

	doc := api.Document()
	if doc.Components == nil || doc.Components.Examples["merchant-live"] == nil {
		t.Fatalf("expected example to be registered")
	}
	if doc.Components.Examples["merchant-live"].Value == nil || doc.Components.Examples["merchant-live"].Value.Summary != "Live merchant" {
		t.Fatalf("unexpected example payload: %+v", doc.Components.Examples["merchant-live"])
	}
}

func TestAPIRegisterSecurityScheme(t *testing.T) {
	api := New()

	err := api.RegisterSecurityScheme("merchantApiKey", &SecuritySchemeOrReference{
		Value: &SecurityScheme{
			Type: "apiKey",
			Name: "X-API-Key",
			In:   "header",
		},
	})
	if err != nil {
		t.Fatalf("register security scheme: %v", err)
	}

	doc := api.Document()
	if doc.Components == nil || doc.Components.SecuritySchemes["merchantApiKey"] == nil {
		t.Fatalf("expected security scheme to be registered")
	}
	if doc.Components.SecuritySchemes["merchantApiKey"].Value == nil || doc.Components.SecuritySchemes["merchantApiKey"].Value.Type != "apiKey" {
		t.Fatalf("unexpected security scheme payload: %+v", doc.Components.SecuritySchemes["merchantApiKey"])
	}
}

func TestAPIJSONIncludesExampleReferences(t *testing.T) {
	api := New()
	if err := api.RegisterExample("merchant-live", InlineExample(&Example{
		Summary: "Live merchant",
		Value: map[string]any{
			"id":   "mrc_live_001",
			"name": "The Base",
		},
	})); err != nil {
		t.Fatalf("register example: %v", err)
	}

	err := api.AddOperation(http.MethodGet, "/merchants/{id}", Operation{
		OperationID: "getMerchant",
		Parameters: []ParameterOrReference{
			{
				Value: &Parameter{
					Name:     "id",
					In:       "path",
					Required: true,
					Schema:   StringSchema(),
					Examples: map[string]*ExampleOrReference{
						"live": ExampleRef("merchant-live"),
					},
				},
			},
		},
		Responses: map[string]ResponseOrReference{
			"200": {
				Value: &Response{
					Description: "ok",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema("Merchant"),
							Examples: map[string]*ExampleOrReference{
								"live": ExampleRef("merchant-live"),
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("add operation: %v", err)
	}

	body, err := api.JSON()
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, "\"examples\"") || !strings.Contains(got, "#/components/examples/merchant-live") {
		t.Fatalf("expected example refs in json: %s", got)
	}
}

func TestWithDocStyle(t *testing.T) {
	api := New(WithDocStyle(DocsBase))
	if api.docsProvider != DocsBase {
		t.Fatalf("unexpected docs provider: %q", api.docsProvider)
	}
}

func TestCanonicalPath(t *testing.T) {
	if got := CanonicalPath("/users/:id/files/*path"); got != "/users/{id}/files/{path}" {
		t.Fatalf("unexpected canonical path: %q", got)
	}
}

func TestCanonicalPathSupportsIrisTemplates(t *testing.T) {
	if got := CanonicalPath("/users/{id:int}/posts/{slug:string}"); got != "/users/{id}/posts/{slug}" {
		t.Fatalf("unexpected iris canonical path: %q", got)
	}
}
