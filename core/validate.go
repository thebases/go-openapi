package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	openAPIVersionPattern = regexp.MustCompile(`^3\.[0-2]\.\d+$`)
	responseKeyPattern    = regexp.MustCompile(`^(default|[1-5]XX|[1-5]\d\d)$`)
	pathTemplatePattern   = regexp.MustCompile(`\{([^{}]+)\}`)
)

// Validate checks the document against the structural OAS rules this library
// can violate. All problems are reported together (errors.Join), in a
// deterministic order (sorted paths, spec method order).
func (d *Document) Validate() error {
	var errs []error
	if !openAPIVersionPattern.MatchString(d.OpenAPI) {
		errs = append(errs, fmt.Errorf("unsupported OpenAPI version %q; expected 3.0.x, 3.1.x, or 3.2.x", d.OpenAPI))
	}
	if strings.TrimSpace(d.Info.Title) == "" {
		errs = append(errs, errors.New("info.title is required"))
	}
	if strings.TrimSpace(d.Info.Version) == "" {
		errs = append(errs, errors.New("info.version is required"))
	}
	if len(d.Paths) == 0 && len(d.Webhooks) == 0 && !hasComponents(d.Components) {
		errs = append(errs, errors.New("document must contain at least one of paths, webhooks, or components"))
	}

	operationIDs := map[string]string{}
	paths := make([]string, 0, len(d.Paths))
	for path := range d.Paths {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		errs = append(errs, d.validatePathItem(path, d.Paths[path], operationIDs)...)
	}

	errs = append(errs, validateComponentNames(d.Components)...)
	errs = append(errs, d.validateRefs()...)
	return errors.Join(errs...)
}

// ValidateDocument validates d; a nil document is an error.
func ValidateDocument(d *Document) error {
	if d == nil {
		return errors.New("openapi: document is nil")
	}
	return d.Validate()
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

func (d *Document) validatePathItem(path string, item *PathItem, operationIDs map[string]string) []error {
	if !strings.HasPrefix(path, "/") {
		return []error{fmt.Errorf("path %q must start with /", path)}
	}
	if item == nil {
		return []error{fmt.Errorf("path %q has nil PathItem", path)}
	}

	var errs []error
	for _, method := range pathItemMethods {
		slot, _ := item.operationSlot(method)
		operation := *slot
		if operation == nil {
			continue
		}
		where := method + " " + path

		if len(operation.Responses) == 0 {
			errs = append(errs, fmt.Errorf("%s has no responses", where))
		}
		for code := range operation.Responses {
			if !responseKeyPattern.MatchString(code) {
				errs = append(errs, fmt.Errorf("%s: invalid response key %q", where, code))
			}
		}

		if operation.OperationID != "" {
			if previous, exists := operationIDs[operation.OperationID]; exists {
				errs = append(errs, fmt.Errorf("duplicate operationId %q used by %s and %s", operation.OperationID, previous, where))
			} else {
				operationIDs[operation.OperationID] = where
			}
		}

		errs = append(errs, d.validateParameters(path, where, item.Parameters, operation.Parameters)...)
	}
	return errs
}

// validateParameters checks the merged path-item and operation parameters:
// no duplicate (name, in), path params are required and present in the
// template, and every template variable is declared.
func (d *Document) validateParameters(path, where string, itemParams, opParams []ParameterOrReference) []error {
	var errs []error
	declared := map[string]bool{}
	seen := map[string]bool{}

	for _, parameter := range append(slices.Clone(itemParams), opParams...) {
		value := d.resolveParameter(parameter)
		if value == nil {
			continue
		}
		key := value.In + ":" + value.Name
		if seen[key] {
			errs = append(errs, fmt.Errorf("%s: duplicate %s parameter %q", where, value.In, value.Name))
		}
		seen[key] = true

		if value.In != "path" {
			continue
		}
		declared[value.Name] = true
		if !value.Required {
			errs = append(errs, fmt.Errorf("path parameter %q must be required", value.Name))
		}
		if !strings.Contains(path, "{"+value.Name+"}") {
			errs = append(errs, fmt.Errorf("path parameter %q is absent from %q", value.Name, path))
		}
	}

	for _, match := range pathTemplatePattern.FindAllStringSubmatch(path, -1) {
		if !declared[match[1]] {
			errs = append(errs, fmt.Errorf("%s: path template variable {%s} has no path parameter", where, match[1]))
		}
	}
	return errs
}

// resolveParameter follows a local components.parameters reference; a
// dangling one is reported by validateRefs, so it resolves to nil here.
func (d *Document) resolveParameter(parameter ParameterOrReference) *Parameter {
	if parameter.Ref == "" {
		return parameter.Value
	}
	name, ok := strings.CutPrefix(parameter.Ref, "#/components/parameters/")
	if !ok || d.Components == nil {
		return nil
	}
	if target := d.Components.Parameters[name]; target != nil {
		return target.Value
	}
	return nil
}

func validateComponentNames(c *Components) []error {
	if c == nil {
		return nil
	}
	var names []string
	for name := range c.Schemas {
		names = append(names, name)
	}
	for name := range c.Parameters {
		names = append(names, name)
	}
	for name := range c.Responses {
		names = append(names, name)
	}
	for name := range c.Examples {
		names = append(names, name)
	}
	for name := range c.RequestBodies {
		names = append(names, name)
	}
	for name := range c.SecuritySchemes {
		names = append(names, name)
	}
	slices.Sort(names)

	var errs []error
	for _, name := range names {
		if !componentNamePattern.MatchString(name) {
			errs = append(errs, fmt.Errorf("%w: %q", ErrInvalidComponentName, name))
		}
	}
	return errs
}

// validateRefs reports every local "#/..." $ref that does not resolve. It
// walks the marshaled JSON so references in any object, including user
// extensions and callbacks, are covered.
func (d *Document) validateRefs() []error {
	raw, err := json.Marshal(d)
	if err != nil {
		return []error{fmt.Errorf("marshal document: %w", err)}
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return []error{fmt.Errorf("unmarshal document: %w", err)}
	}

	var refs []string
	collectRefs(root, &refs)
	slices.Sort(refs)
	refs = slices.Compact(refs)

	var errs []error
	for _, ref := range refs {
		if pointer, local := strings.CutPrefix(ref, "#"); local && !resolvesPointer(root, pointer) {
			errs = append(errs, fmt.Errorf("unresolved $ref %q", ref))
		}
	}
	return errs
}

func collectRefs(node any, refs *[]string) {
	switch value := node.(type) {
	case map[string]any:
		if ref, ok := value["$ref"].(string); ok {
			*refs = append(*refs, ref)
		}
		for _, child := range value {
			collectRefs(child, refs)
		}
	case []any:
		for _, child := range value {
			collectRefs(child, refs)
		}
	}
}

// resolvesPointer evaluates an RFC 6901 JSON pointer ("/components/schemas/X").
func resolvesPointer(root any, pointer string) bool {
	if pointer == "" {
		return true
	}
	current := root
	for _, token := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		token = strings.NewReplacer("~1", "/", "~0", "~").Replace(token)
		object, ok := current.(map[string]any)
		if !ok {
			return false
		}
		if current, ok = object[token]; !ok {
			return false
		}
	}
	return true
}
