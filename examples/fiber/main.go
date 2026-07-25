package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	app := fiber.New()
	api := openapi.New(
		openapi.WithTitle("Fiber Example API"),
		openapi.WithVersion("1.0.0"),
	)

	err := openapi.Fiber.GET(app, api, "/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}, func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := openapi.Fiber.MountDocs(app, api, "/docs", "/openapi.json", openapi.DocsConfig{Provider: openapi.DocsSwagger, Title: "Fiber Example API"}); err != nil {
		log.Fatal(err)
	}

	log.Fatal(app.Listen(":3000"))
}
