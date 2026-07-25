package main

import (
	"log"

	"github.com/gin-gonic/gin"
	api "github.com/thebases/go-openapi/integrations/gin"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	router := gin.New()
	docs := openapi.New(
		openapi.WithTitle("Gin Example API"),
		openapi.WithDescription("descriptions/api.md"),
		openapi.WithVersion("1.0.0"),
		openapi.WithDocStyle(openapi.DocsSwagger),
	)
	if err := docs.RegisterSchema("Merchant", openapi.InlineSchema(&openapi.Schema{
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

	err := api.GET(router, docs, openapi.Route("/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Description: "descriptions/merchant-operation.md",
		Parameters: []openapi.ParameterOrReference{
			{Value: &openapi.Parameter{Name: "id", In: "path", Required: true, Description: "descriptions/merchant-id.md", Schema: openapi.StringSchema()}},
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("descriptions/merchant-response.md", openapi.RefSchema("Merchant")),
		},
	}), func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.Param("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(router.Run(":3000"))
}
