package fiberadapter

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	spec "github.com/thebases/go-openapi/openapi"
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
	Security      []spec.SecurityRequirement
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

	// Keep the runtime handler registration colocated with the OpenAPI metadata
	// registration so the generated spec stays aligned with the actual Fiber route.
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

func decodeInput[I any](c fiber.Ctx, target *I, config OperationConfig) error {
	if config.RequestBody {
		if err := c.Bind().Body(target); err != nil {
			return &HTTPError{
				Status:  fiber.StatusBadRequest,
				Code:    "invalid_body",
				Message: "Request body is invalid",
				Cause:   err,
			}
		}
		return nil
	}

	return decodeTaggedFields(c, target)
}

func decodeTaggedFields(c fiber.Ctx, target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("target must point to a struct")
	}

	valueType := value.Type()

	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := valueType.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		var raw string
		switch {
		case fieldType.Tag.Get("path") != "":
			raw = c.Params(fieldType.Tag.Get("path"))
		case fieldType.Tag.Get("query") != "":
			raw = c.Query(fieldType.Tag.Get("query"))
		case fieldType.Tag.Get("header") != "":
			raw = c.Get(fieldType.Tag.Get("header"))
		case fieldType.Tag.Get("cookie") != "":
			raw = c.Cookies(fieldType.Tag.Get("cookie"))
		default:
			continue
		}

		if raw == "" {
			continue
		}

		if err := setFromString(fieldValue, raw); err != nil {
			return fmt.Errorf("decode field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

func setFromString(value reflect.Value, raw string) error {
	if value.Kind() == reflect.Pointer {
		value.Set(reflect.New(value.Type().Elem()))
		return setFromString(value.Elem(), raw)
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, value.Type().Bits())
		if err != nil {
			return err
		}
		value.SetFloat(parsed)
	case reflect.Slice:
		parts := strings.Split(raw, ",")
		result := reflect.MakeSlice(value.Type(), len(parts), len(parts))
		for i, part := range parts {
			if err := setFromString(result.Index(i), strings.TrimSpace(part)); err != nil {
				return err
			}
		}
		value.Set(result)
	default:
		return fmt.Errorf("unsupported parameter type %s", value.Type())
	}

	return nil
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
