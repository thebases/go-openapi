package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	openapi "github.com/thebases/go-openapi/openapi"
)

func main() {
	router := chi.NewRouter()
	api := openapi.New(
		openapi.WithTitle("Chi Example API"),
		openapi.WithVersion("1.0.0"),
	)

	err := openapi.Chi.GET(router, api, "/merchants/{id}", openapi.Operation{
		OperationID: "getMerchant",
		Summary:     "Get merchant",
		Parameters: []openapi.ParameterOrReference{
			openapi.PathParameter("id", openapi.StringSchema()),
		},
		Responses: map[string]openapi.ResponseOrReference{
			"200": openapi.JSONResponse("Merchant returned", openapi.RefSchema("Merchant")),
		},
	}, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   chi.URLParam(r, "id"),
			"name": "The Base",
		})
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := openapi.Chi.MountDocs(router, api, "/docs", "/openapi.json", openapi.DocsConfig{Provider: openapi.DocsSwagger, Title: "Chi Example API"}); err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(":3000", router))
}
