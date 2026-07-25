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
		openapi.WithDocStyle(openapi.DocsScalar),
	)
	if err := doc.RegisterSchema("Merchant", openapi.InlineSchema(&openapi.Schema{
		Type:        "object",
		Description: "descriptions/merchant-schema.md",
		Properties: map[string]*openapi.SchemaOrReference{
			"id":   openapi.StringSchema(),
			"name": openapi.StringSchema(),
		},
		Required: []string{"id", "name"},
	})); err != nil {
		log.Fatal(err)
	}

	err := api.GET(app, doc, openapi.Route("/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Description: "descriptions/merchant-operation.md",
		Parameters: []openapi.ParameterOrReference{
			{Value: &openapi.Parameter{Name: "id", In: "path", Required: true, Description: "descriptions/merchant-id.md", Schema: openapi.StringSchema()}},
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("descriptions/merchant-response.md", openapi.RefSchema("Merchant")),
		},
	}), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(app.Listen(":3000"))
}
