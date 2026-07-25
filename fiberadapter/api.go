package fiberadapter

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/thebases/go-openapi/docs"
	spec "github.com/thebases/go-openapi/openapi"
	"github.com/thebases/go-openapi/reflector"
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
	document  *spec.Document
	reflector *reflector.Reflector
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
		config.JSONPath = "/docs/openapi.json"
	}
	config.JSONPath = docs.NormalizeRoutePath(config.JSONPath)

	api := &API{
		app:       app,
		config:    config,
		reflector: reflector.New(),
		document: &spec.Document{
			OpenAPI: config.OpenAPI,
			Info: spec.Info{
				Title:       config.Title,
				Description: config.Description,
				Version:     config.Version,
			},
			Paths: make(map[string]*spec.PathItem),
			Components: &spec.Components{
				Schemas:         make(map[string]*spec.SchemaOrReference),
				SecuritySchemes: make(map[string]*spec.SecuritySchemeOrReference),
			},
		},
	}

	// Register docs routes once so the adapter owns the Fiber wiring and the
	// document model stays reusable for other routers.
	docs.Register(app, config.JSONPath, api.Document)
	return api
}

func (a *API) Document() *spec.Document {
	return a.document
}

func (a *API) AddBearerAuth(name, description string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.document.Components.SecuritySchemes == nil {
		a.document.Components.SecuritySchemes = make(map[string]*spec.SecuritySchemeOrReference)
	}

	a.document.Components.SecuritySchemes[name] = &spec.SecuritySchemeOrReference{
		Value: &spec.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  description,
		},
	}
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

	operation := &spec.Operation{
		OperationID: config.ID,
		Summary:     config.Summary,
		Description: config.Description,
		Tags:        config.Tags,
		Deprecated:  config.Deprecated,
		Security:    config.Security,
		Responses:   make(map[string]spec.ResponseOrReference),
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

		operation.RequestBody = &spec.RequestBodyOrReference{
			Value: &spec.RequestBody{
				Required: true,
				Content: map[string]spec.MediaType{
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

		operation.Responses[strconv.Itoa(status)] = spec.ResponseOrReference{
			Value: &spec.Response{
				Description: http.StatusText(status),
				Content: map[string]spec.MediaType{
					"application/json": {Schema: responseSchema},
				},
			},
		}
	} else {
		operation.Responses[strconv.Itoa(status)] = spec.ResponseOrReference{
			Value: &spec.Response{Description: http.StatusText(status)},
		}
	}

	pathKey := spec.FiberPathToOpenAPI(path)
	pathItem := a.document.Paths[pathKey]
	if pathItem == nil {
		pathItem = &spec.PathItem{}
		a.document.Paths[pathKey] = pathItem
	}

	if err := spec.SetOperation(pathItem, method, operation); err != nil {
		return err
	}

	for name, schema := range a.reflector.Components {
		a.document.Components.Schemas[name] = schema
	}

	return nil
}
