package openapi

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type referenceTestNested struct {
	ID string `json:"id" format:"uuid"`
}

type referenceTestRequest struct {
	Nested *referenceTestNested `json:"nested,omitempty" description:"Nested payload"`
}

type referenceTestResponse struct {
	Data *referenceTestNested `json:"data,omitempty"`
}

func TestSchemaRefMarshal_AllowsTypedNilValue(t *testing.T) {
	schema := &SchemaOrReference{Ref: "#/components/schemas/Test"}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema ref: %v", err)
	}

	if string(data) != `{"$ref":"#/components/schemas/Test"}` {
		t.Fatalf("unexpected schema ref json: %s", data)
	}
}

func TestDocumentMarshal_WithReferencedFieldOverlays(t *testing.T) {
	app := fiber.New()
	api := New(app, Config{
		Title:   "Test API",
		Version: "1.0.0",
	})

	POST(api, "/items", OperationConfig{ID: "createItem"}, func(ctx fiber.Ctx, input referenceTestRequest) (referenceTestResponse, error) {
		return referenceTestResponse{}, nil
	})

	// This regression test guards the route-serving path because /docs/openapi.json
	// marshals the entire document on demand.
	if _, err := json.Marshal(api.Document()); err != nil {
		t.Fatalf("marshal document: %v", err)
	}
}
