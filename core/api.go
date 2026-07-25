package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrInvalidPath       = errors.New("openapi: path must start with /")
	ErrUnsupportedMethod = errors.New("openapi: unsupported HTTP method")
	ErrDuplicateRoute    = errors.New("openapi: operation already registered")
	ErrMissingResponses  = errors.New("openapi: operation must define at least one response")
)

// API coordinates concurrent-safe mutations of the OpenAPI document.
type API struct {
	mu  sync.RWMutex
	doc Document
}

func New(options ...Option) *API {
	api := &API{
		doc: Document{
			OpenAPI: "3.0.3",
			Info: Info{
				Title:   "API",
				Version: "0.0.0",
			},
			Paths: map[string]*PathItem{},
			Components: Components{
				Schemas: map[string]*Schema{},
			},
		},
	}

	for _, option := range options {
		option(api)
	}

	return api
}

func (api *API) AddOperation(method, path string, operation Operation) error {
	if !strings.HasPrefix(path, "/") {
		return ErrInvalidPath
	}
	if len(operation.Responses) == 0 {
		return ErrMissingResponses
	}

	method = strings.ToUpper(method)

	api.mu.Lock()
	defer api.mu.Unlock()

	item := api.doc.Paths[path]
	if item == nil {
		item = &PathItem{}
		api.doc.Paths[path] = item
	}

	target, err := operationSlot(item, method)
	if err != nil {
		return err
	}
	if *target != nil {
		return fmt.Errorf("%w: %s %s", ErrDuplicateRoute, method, path)
	}

	// Store a copy so later caller mutations cannot race with the shared document.
	copy := operation
	*target = &copy

	return nil
}

func (api *API) RegisterSchema(name string, schema *Schema) error {
	if strings.TrimSpace(name) == "" || schema == nil {
		return errors.New("openapi: schema name and value are required")
	}

	api.mu.Lock()
	defer api.mu.Unlock()

	if _, exists := api.doc.Components.Schemas[name]; exists {
		return fmt.Errorf("openapi: schema %q already registered", name)
	}

	api.doc.Components.Schemas[name] = schema
	return nil
}

func (api *API) Document() Document {
	api.mu.RLock()
	defer api.mu.RUnlock()

	// JSON round-trip keeps the snapshot isolated without exposing shared pointers.
	raw, _ := json.Marshal(api.doc)
	var result Document
	_ = json.Unmarshal(raw, &result)
	return result
}

func (api *API) JSON() ([]byte, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return json.MarshalIndent(api.doc, "", "  ")
}

func operationSlot(item *PathItem, method string) (**Operation, error) {
	switch method {
	case "GET":
		return &item.Get, nil
	case "PUT":
		return &item.Put, nil
	case "POST":
		return &item.Post, nil
	case "DELETE":
		return &item.Delete, nil
	case "OPTIONS":
		return &item.Options, nil
	case "HEAD":
		return &item.Head, nil
	case "PATCH":
		return &item.Patch, nil
	case "TRACE":
		return &item.Trace, nil
	default:
		return nil, ErrUnsupportedMethod
	}
}
