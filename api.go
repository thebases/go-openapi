package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v3"
)

type Config struct {
	Title       string
	Description string
	Version     string
	OpenAPI     string
	JSONPath    string
}

type API struct {
	app       *fiber.App
	config    Config
	document  *Document
	reflector *Reflector
	mu        sync.RWMutex
}

func New(app *fiber.App, config Config) *API {
	if app == nil {
		panic("openapi: nil Fiber app")
	}
	if config.Title == "" {
		config.Title = "API"
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.OpenAPI == "" {
		config.OpenAPI = "3.0.3"
	}
	if config.JSONPath == "" {
		config.JSONPath = "/openapi.json"
	}

	reflector := NewReflector()

	api := &API{
		app:       app,
		config:    config,
		reflector: reflector,
		document: &Document{
			OpenAPI: config.OpenAPI,
			Info: Info{
				Title:       config.Title,
				Description: config.Description,
				Version:     config.Version,
			},
			Paths: make(map[string]*PathItem),
			Components: &Components{
				Schemas:         make(map[string]*SchemaOrReference),
				SecuritySchemes: make(map[string]*SecuritySchemeOrReference),
			},
		},
	}

	api.registerDocumentRoute()
	return api
}

func (a *API) Document() *Document {
	return a.document
}

func (a *API) registerDocumentRoute() {
	a.app.Get(a.config.JSONPath, func(c fiber.Ctx) error {
		a.mu.RLock()
		defer a.mu.RUnlock()

		data, err := json.MarshalIndent(a.document, "", "  ")
		if err != nil {
			return err
		}

		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.Send(data)
	})
}

func (a *API) addOperation(
	method string,
	path string,
	config OperationConfig,
	inputType reflect.Type,
	outputType reflect.Type,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	operation := &Operation{
		OperationID: config.ID,
		Summary:     config.Summary,
		Description: config.Description,
		Tags:        config.Tags,
		Deprecated:  config.Deprecated,
		Security:    config.Security,
		Responses:   make(map[string]ResponseOrReference),
	}

	if config.RequestBody {
		requestSchema, err := a.reflector.ReflectType(inputType)
		if err != nil {
			return fmt.Errorf("request schema: %w", err)
		}

		contentType := config.ContentType
		if contentType == "" {
			contentType = "application/json"
		}

		operation.RequestBody = &RequestBodyOrReference{
			Value: &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					contentType: {Schema: requestSchema},
				},
			},
		}
	} else {
		parameters, err := a.parametersFor(inputType)
		if err != nil {
			return fmt.Errorf("request parameters: %w", err)
		}
		operation.Parameters = parameters
	}

	status := config.SuccessStatus
	if status == 0 {
		status = defaultSuccessStatus(method)
	}

	if status != http.StatusNoContent {
		responseSchema, err := a.reflector.ReflectType(outputType)
		if err != nil {
			return fmt.Errorf("response schema: %w", err)
		}

		operation.Responses[strconv.Itoa(status)] = ResponseOrReference{
			Value: &Response{
				Description: http.StatusText(status),
				Content: map[string]MediaType{
					"application/json": {Schema: responseSchema},
				},
			},
		}
	} else {
		operation.Responses[strconv.Itoa(status)] = ResponseOrReference{
			Value: &Response{Description: http.StatusText(status)},
		}
	}

	pathKey := openAPIPath(path)
	pathItem := a.document.Paths[pathKey]
	if pathItem == nil {
		pathItem = &PathItem{}
		a.document.Paths[pathKey] = pathItem
	}

	if err := setOperation(pathItem, method, operation); err != nil {
		return err
	}

	for name, schema := range a.reflector.Components {
		a.document.Components.Schemas[name] = schema
	}

	return nil
}
