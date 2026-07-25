package openapi

import (
	"fmt"
	"regexp"
	"strings"
)

var openAPIVersionPattern = regexp.MustCompile(`^3\.0\.\d+$`)

func (d *Document) Validate() error {
	if !openAPIVersionPattern.MatchString(d.OpenAPI) {
		return fmt.Errorf("unsupported OpenAPI version %q; expected 3.0.x", d.OpenAPI)
	}
	if strings.TrimSpace(d.Info.Title) == "" {
		return fmt.Errorf("info.title is required")
	}
	if strings.TrimSpace(d.Info.Version) == "" {
		return fmt.Errorf("info.version is required")
	}
	if d.Paths == nil {
		return fmt.Errorf("paths is required")
	}

	operationIDs := map[string]string{}

	for path, item := range d.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path %q must start with /", path)
		}
		if item == nil {
			return fmt.Errorf("path %q has nil PathItem", path)
		}

		if err := validatePathItem(path, item, operationIDs); err != nil {
			return err
		}
	}

	return nil
}

func validatePathItem(path string, item *PathItem, operationIDs map[string]string) error {
	operations := map[string]*Operation{
		"GET": item.Get, "PUT": item.Put, "POST": item.Post,
		"DELETE": item.Delete, "OPTIONS": item.Options,
		"HEAD": item.Head, "PATCH": item.Patch, "TRACE": item.Trace,
	}

	for method, operation := range operations {
		if operation == nil {
			continue
		}

		if len(operation.Responses) == 0 {
			return fmt.Errorf("%s %s has no responses", method, path)
		}

		if operation.OperationID != "" {
			if previous, exists := operationIDs[operation.OperationID]; exists {
				return fmt.Errorf(
					"duplicate operationId %q used by %s and %s %s",
					operation.OperationID, previous, method, path,
				)
			}
			operationIDs[operation.OperationID] = method + " " + path
		}

		for _, parameter := range operation.Parameters {
			if parameter.Value == nil {
				continue
			}

			if parameter.Value.In == "path" && !parameter.Value.Required {
				return fmt.Errorf("path parameter %q must be required", parameter.Value.Name)
			}
			if parameter.Value.In == "path" &&
				!strings.Contains(path, "{"+parameter.Value.Name+"}") {
				return fmt.Errorf(
					"path parameter %q is absent from %q",
					parameter.Value.Name, path,
				)
			}
		}
	}

	return nil
}
