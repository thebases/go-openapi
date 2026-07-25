package openapi

import (
	"net/http"
	"testing"
)

func TestAPIAddOperation(t *testing.T) {
	api := New(
		WithTitle("Merchant API"),
		WithVersion("1.0.0"),
	)

	err := api.AddOperation(http.MethodGet, "/merchants/{id}", Operation{
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
