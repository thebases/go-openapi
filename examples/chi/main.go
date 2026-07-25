package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	api "github.com/thebases/go-openapi/integrations/chi"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	router := chi.NewRouter()
	doc := openapi.New(
		openapi.WithTitle("Chi Example API"),
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

	err := api.GET(router, doc, openapi.Route("/merchants/{id}", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}), func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   chi.URLParam(r, "id"),
			"name": "The Base",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(":3000", router))
}
