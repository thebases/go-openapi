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
		openapi.WithVersion("1.0.0"),
		openapi.WithDocStyle(openapi.DocsSwagger),
	)

	err := api.GET(router, docs, openapi.Route("/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}), func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.Param("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(router.Run(":3000"))
}
