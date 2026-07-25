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
		openapi.WithVersion("1.0.0"),
		openapi.WithDocStyle(openapi.DocsSwagger),
	)
	if err := doc.RegisterSchema("Merchant", openapi.InlineSchema(&openapi.Schema{
		Type: "object",
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
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(app.Listen(":3000"))
}
