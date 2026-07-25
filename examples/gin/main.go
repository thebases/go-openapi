package main

import (
	"log"

	"github.com/gin-gonic/gin"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	router := gin.New()
	api := openapi.New(
		openapi.WithTitle("Gin Example API"),
		openapi.WithVersion("1.0.0"),
	)

	err := openapi.Gin.GET(router, api, "/merchants/:id", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}, func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.Param("id"), "name": "The Base"})
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := openapi.Gin.MountDocs(router, api, "/docs", "/openapi.json", openapi.DocsConfig{Provider: openapi.DocsSwagger, Title: "Gin Example API"}); err != nil {
		log.Fatal(err)
	}

	log.Fatal(router.Run(":3000"))
}
