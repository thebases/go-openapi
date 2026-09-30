package core

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// updateGolden rewrites testdata/golden/*.json: go test ./core -run Golden -update
var updateGolden = flag.Bool("update", false, "rewrite golden OpenAPI documents")

type goldenAddress struct {
	City string `json:"city" example:"Hanoi"`
}

type goldenMerchant struct {
	ID        int64          `json:"id" readOnly:"true" example:"42"`
	Name      string         `json:"name" minLength:"1" maxLength:"64"`
	Status    string         `json:"status" enum:"active,suspended" default:"active"`
	Balance   float64        `json:"balance" minimum:"0"`
	Address   *goldenAddress `json:"address"`
	Tags      []string       `json:"tags,omitempty"`
	Parent    *string        `json:"parentId"`
	Attribute map[string]int `json:"attributes,omitempty"`
}

// goldenAPI exercises every feature that differs between spec versions:
// nullable scalars and $refs, examples, extensions, and security.
func goldenAPI(t *testing.T, version SpecVersion) *API {
	t.Helper()
	api := New(
		WithOpenAPIVersion(version),
		WithTitle("Golden API"),
		WithVersion("1.0.0"),
		WithServer("https://api.example.com", "Production"),
	)

	reflector := NewReflector()
	ref, err := reflector.ReflectType(reflect.TypeOf(goldenMerchant{}))
	if err != nil {
		t.Fatal(err)
	}
	for name, schema := range reflector.Components {
		if err := api.RegisterSchema(name, schema); err != nil {
			t.Fatal(err)
		}
	}
	if err := api.RegisterSecurityScheme("apiKey", &SecuritySchemeOrReference{Value: &SecurityScheme{
		Type: "apiKey", Name: "X-API-Key", In: "header",
	}}); err != nil {
		t.Fatal(err)
	}

	if err := api.AddOperation(http.MethodGet, "/merchants/{id}", Operation{
		OperationID: "getMerchant",
		Tags:        []string{"merchants"},
		Parameters:  []ParameterOrReference{PathParameter("id", IntegerSchema("int64"))},
		Responses:   map[string]ResponseOrReference{"200": JSONResponse("Merchant", ref)},
		Security:    []SecurityRequirement{{"apiKey": {}}},
		Extensions:  map[string]any{"x-rate-limit": 100},
	}); err != nil {
		t.Fatal(err)
	}
	return api
}

func TestGoldenDocuments(t *testing.T) {
	for _, version := range []SpecVersion{Version30, Version31, Version32} {
		t.Run(version.String(), func(t *testing.T) {
			api := goldenAPI(t, version)
			doc, err := api.Snapshot()
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.Validate(); err != nil {
				t.Fatalf("golden document is invalid:\n%v", err)
			}

			got, err := api.JSON()
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("testdata", "golden", "openapi-"+version.String()+".json")
			if *updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if string(got) != string(want) {
				t.Fatalf("%s differs from %s; review the diff and rerun with -update if intended\n%s", version, path, got)
			}
		})
	}
}
