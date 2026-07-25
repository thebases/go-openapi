package openapi

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type describedMerchant struct {
	ID string `json:"id" description:"schema.md"`
}

func TestAPIDescriptionMarkdownFilesResolveAcrossDocument(t *testing.T) {
	dir := t.TempDir()
	writeDescriptionFixtures(t, dir, map[string]string{
		"info.md":      "Info description\n",
		"server.md":    "Server description\n",
		"tag.md":       "Tag description\n",
		"external.md":  "External docs description\n",
		"path.md":      "Path description\n",
		"parameter.md": "Parameter description\n",
		"operation.md": "Operation description\n",
		"request.md":   "Request body description\n",
		"response.md":  "Response description\n",
		"schema.md":    "Schema description\n",
		"security.md":  "Security scheme description\n",
	})

	restore := withWorkingDirectory(t, dir)
	defer restore()

	api := New(
		WithDescription("info.md"),
		WithServer("https://api.example.com", "server.md"),
	)
	api.doc.Tags = []Tag{{
		Name:         "Merchants",
		Description:  "tag.md",
		ExternalDocs: &ExternalDocumentation{URL: "https://docs.example.com", Description: "external.md"},
	}}
	api.doc.Paths["/merchants"] = &PathItem{
		Description: "path.md",
		Parameters: []ParameterOrReference{{
			Value: &Parameter{
				Name:        "merchantId",
				In:          "query",
				Description: "parameter.md",
				Schema:      StringSchema(),
			},
		}},
		Post: &Operation{
			Description: "operation.md",
			RequestBody: &RequestBodyOrReference{
				Value: &RequestBody{
					Description: "request.md",
					Content: map[string]MediaType{
						"application/json": {
							Schema: InlineSchema(&Schema{
								Type:        "object",
								Description: "schema.md",
							}),
						},
					},
				},
			},
			Responses: map[string]ResponseOrReference{
				"200": {
					Value: &Response{
						Description: "response.md",
					},
				},
			},
			Security: []SecurityRequirement{{"bearerAuth": []string{}}},
		},
	}
	api.doc.Components.SecuritySchemes = map[string]*SecuritySchemeOrReference{
		"bearerAuth": &SecuritySchemeOrReference{
			Value: &SecurityScheme{
				Type:        "http",
				Scheme:      "bearer",
				Description: "security.md",
			},
		},
	}

	ref := NewReflector()
	_, err := ref.ReflectType(reflectType[describedMerchant]())
	if err != nil {
		t.Fatalf("reflect described merchant: %v", err)
	}
	for name, component := range ref.Components {
		if err := api.RegisterSchema(name, component); err != nil {
			t.Fatalf("register reflected component %s: %v", name, err)
		}
	}

	doc := api.Document()
	if doc.Info.Description != "Info description\n" {
		t.Fatalf("unexpected info description: %q", doc.Info.Description)
	}
	if doc.Servers[0].Description != "Server description\n" {
		t.Fatalf("unexpected server description: %q", doc.Servers[0].Description)
	}
	if doc.Tags[0].Description != "Tag description\n" {
		t.Fatalf("unexpected tag description: %q", doc.Tags[0].Description)
	}
	if doc.Tags[0].ExternalDocs.Description != "External docs description\n" {
		t.Fatalf("unexpected external docs description: %q", doc.Tags[0].ExternalDocs.Description)
	}
	if doc.Paths["/merchants"].Description != "Path description\n" {
		t.Fatalf("unexpected path description: %q", doc.Paths["/merchants"].Description)
	}
	if doc.Paths["/merchants"].Parameters[0].Value.Description != "Parameter description\n" {
		t.Fatalf("unexpected parameter description: %q", doc.Paths["/merchants"].Parameters[0].Value.Description)
	}
	if doc.Paths["/merchants"].Post.Description != "Operation description\n" {
		t.Fatalf("unexpected operation description: %q", doc.Paths["/merchants"].Post.Description)
	}
	if doc.Paths["/merchants"].Post.RequestBody.Value.Description != "Request body description\n" {
		t.Fatalf("unexpected request body description: %q", doc.Paths["/merchants"].Post.RequestBody.Value.Description)
	}
	if doc.Paths["/merchants"].Post.RequestBody.Value.Content["application/json"].Schema.Value.Description != "Schema description\n" {
		t.Fatalf("unexpected inline schema description: %q", doc.Paths["/merchants"].Post.RequestBody.Value.Content["application/json"].Schema.Value.Description)
	}
	if doc.Paths["/merchants"].Post.Responses["200"].Value.Description != "Response description\n" {
		t.Fatalf("unexpected response description: %q", doc.Paths["/merchants"].Post.Responses["200"].Value.Description)
	}
	if doc.Components.SecuritySchemes["bearerAuth"].Value.Description != "Security scheme description\n" {
		t.Fatalf("unexpected security scheme description: %q", doc.Components.SecuritySchemes["bearerAuth"].Value.Description)
	}
	if doc.Components.Schemas["describedMerchant"].Value.Properties["id"].Value.Description != "Schema description\n" {
		t.Fatalf("unexpected reflected schema description: %q", doc.Components.Schemas["describedMerchant"].Value.Properties["id"].Value.Description)
	}
}

func TestAPIJSONReturnsErrorForMissingMarkdownDescription(t *testing.T) {
	restore := withWorkingDirectory(t, t.TempDir())
	defer restore()

	api := New(WithDescription("missing.md"))

	_, err := api.JSON()
	if err == nil {
		t.Fatal("expected JSON to fail for missing markdown description")
	}
	if !strings.Contains(err.Error(), "missing.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func reflectType[T any]() reflect.Type {
	var zero *T
	return reflect.TypeOf(zero).Elem()
}

func withWorkingDirectory(t *testing.T, dir string) func() {
	t.Helper()

	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory to %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(current); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}
}

func writeDescriptionFixtures(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}
