package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	openapi "github.com/thebases/go-openapi"
)

type GetMerchantInput struct {
	ID      string `path:"id" description:"Merchant identifier"`
	Verbose bool   `query:"verbose" description:"Include additional information"`
}

type CreateMerchantInput struct {
	Name    string `json:"name" minLength:"1" maxLength:"200"`
	TaxCode string `json:"taxCode" minLength:"10" maxLength:"14"`
}

type Merchant struct {
	ID      string  `json:"id" format:"uuid"`
	Name    string  `json:"name" minLength:"1" maxLength:"200"`
	Status  string  `json:"status" enum:"active,inactive,suspended"`
	Website *string `json:"website,omitempty" nullable:"true"`
}

func main() {
	app := fiber.New()

	api := openapi.New(app, openapi.Config{
		Title:       "Merchant API",
		Description: "Merchant management service",
		Version:     "1.0.0",
		OpenAPI:     "3.0.3",
	})

	api.AddBearerAuth("BearerAuth", "JWT access token")

	openapi.GET(
		api,
		"/merchants/:id",
		openapi.OperationConfig{
			ID:      "getMerchant",
			Summary: "Get merchant",
			Tags:    []string{"Merchants"},
			Security: []openapi.SecurityRequirement{
				{"BearerAuth": {}},
			},
		},
		func(ctx fiber.Ctx, input GetMerchantInput) (Merchant, error) {
			return Merchant{
				ID:     input.ID,
				Name:   "The Base Store",
				Status: "active",
			}, nil
		},
	)

	openapi.POST(
		api,
		"/merchants",
		openapi.OperationConfig{
			ID:      "createMerchant",
			Summary: "Create merchant",
			Tags:    []string{"Merchants"},
		},
		func(ctx fiber.Ctx, input CreateMerchantInput) (Merchant, error) {
			return Merchant{
				ID:     "merchant-001",
				Name:   input.Name,
				Status: "active",
			}, nil
		},
	)

	if err := api.Document().Validate(); err != nil {
		log.Fatal(err)
	}

	log.Fatal(app.Listen(":3000"))
}
