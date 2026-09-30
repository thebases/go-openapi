package core

import (
	"errors"
	"regexp"
)

// Sentinel errors returned (wrapped with context) by core. Match them with
// errors.Is.
var (
	ErrInvalidPath          = errors.New("openapi: path must start with /")
	ErrUnsupportedMethod    = errors.New("openapi: unsupported HTTP method")
	ErrDuplicateRoute       = errors.New("openapi: operation already registered")
	ErrMissingResponses     = errors.New("openapi: operation must define at least one response")
	ErrNoHandler            = errors.New("openapi: route needs at least one handler")
	ErrDuplicateComponent   = errors.New("openapi: component already registered")
	ErrInvalidComponentName = errors.New("openapi: component name must match ^[a-zA-Z0-9._-]+$")
	ErrUnsupportedVersion   = errors.New("openapi: unsupported OpenAPI version")
	ErrInvalidDescription   = errors.New("openapi: invalid markdown description path")
	ErrInvalidTag           = errors.New("openapi: invalid struct tag value")
)

// componentNamePattern is the OAS rule for keys under components.*.
var componentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
