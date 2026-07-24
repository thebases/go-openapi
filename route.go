package openapi

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type Handler[I, O any] func(ctx fiber.Ctx, input I) (O, error)

type OperationConfig struct {
	ID            string
	Summary       string
	Description   string
	Tags          []string
	Deprecated    bool
	SuccessStatus int
	RequestBody   bool
	ContentType   string
	Security      []SecurityRequirement
}

func GET[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	register(api, fiber.MethodGet, path, config, handler)
}

func POST[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	config.RequestBody = true
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}
	register(api, fiber.MethodPost, path, config, handler)
}

func PUT[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	config.RequestBody = true
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}
	register(api, fiber.MethodPut, path, config, handler)
}

func PATCH[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	config.RequestBody = true
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}
	register(api, fiber.MethodPatch, path, config, handler)
}

func DELETE[I, O any](api *API, path string, config OperationConfig, handler Handler[I, O]) {
	register(api, fiber.MethodDelete, path, config, handler)
}

func register[I, O any](
	api *API,
	method string,
	path string,
	config OperationConfig,
	handler Handler[I, O],
) {
	if api == nil {
		panic("openapi: nil API")
	}
	if handler == nil {
		panic("openapi: nil handler")
	}

	inputType := reflect.TypeOf((*I)(nil)).Elem()
	outputType := reflect.TypeOf((*O)(nil)).Elem()

	if err := api.addOperation(method, path, config, inputType, outputType); err != nil {
		panic(err)
	}

	api.app.Add([]string{method}, path, func(c fiber.Ctx) error {
		var input I

		if err := decodeInput(c, &input, config); err != nil {
			return writeError(c, err)
		}

		output, err := handler(c, input)
		if err != nil {
			return writeError(c, err)
		}

		status := config.SuccessStatus
		if status == 0 {
			status = defaultSuccessStatus(method)
		}

		if status == http.StatusNoContent {
			return c.SendStatus(status)
		}
		return c.Status(status).JSON(output)
	})
}

func defaultSuccessStatus(method string) int {
	switch strings.ToUpper(method) {
	case fiber.MethodPost:
		return http.StatusCreated
	case fiber.MethodDelete:
		return http.StatusNoContent
	default:
		return http.StatusOK
	}
}
