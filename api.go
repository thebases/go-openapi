package openapi

import (
	"github.com/gofiber/fiber/v3"
	"github.com/thebases/go-openapi/fiberadapter"
)

type API = fiberadapter.API

type Config = fiberadapter.Config

func New(app *fiber.App, config Config) *API {
	return fiberadapter.New(app, config)
}
