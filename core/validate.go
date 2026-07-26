package core

import (
	"fmt"
	"regexp"
	"strings"
)

var openAPIVersionPattern = regexp.MustCompile(`^3\.[0-2]\.\d+$`)

func (d *Document) Validate() error {
	if !openAPIVersionPattern.MatchString(d.OpenAPI) {
		return fmt.Errorf("unsupported OpenAPI version %q; expected 3.0.x, 3.1.x, or 3.2.x", d.OpenAPI)
	}
	if strings.TrimSpace(d.Info.Title) == "" {
		return fmt.Errorf("info.title is required")
	}
	if strings.TrimSpace(d.Info.Version) == "" {
		return fmt.Errorf("info.version is required")
	}
	if len(d.Paths) == 0 && len(d.Webhooks) == 0 && !hasComponents(d.Components) {
		return fmt.Errorf("document must contain at least one of paths, webhooks, or components")
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

func hasComponents(c *Components) bool {
	if c == nil {
		return false
	}
	return len(c.Schemas) > 0 || len(c.Responses) > 0 || len(c.Parameters) > 0 ||
		len(c.Examples) > 0 || len(c.RequestBodies) > 0 || len(c.Headers) > 0 ||
		len(c.SecuritySchemes) > 0 || len(c.Links) > 0 || len(c.Callbacks) > 0 ||
		len(c.PathItems) > 0
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
