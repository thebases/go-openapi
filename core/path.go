package core

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// nativeParamPattern matches ":name" / "*name" segments, including Fiber's
	// "<constraint>" and optional "?" suffixes, which OpenAPI cannot express.
	nativeParamPattern = regexp.MustCompile(`([:*])([A-Za-z_][A-Za-z0-9_]*)(?:<[^>]*>)?\??`)
	// irisParamPattern matches Iris "{name}" / "{name:type}" templates.
	irisParamPattern = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_]*)(?::[^{}]+)?\}`)
	// unnamedWildcardPattern matches a bare "*" or Fiber "+" segment (chi, echo,
	// fiber), which would otherwise leak into the doc path as a literal.
	unnamedWildcardPattern = regexp.MustCompile(`/[*+](/|$)`)
)

// FiberPathToOpenAPI converts ":name"/"*name" (Fiber, Gin, Echo, Chi) params
// and unnamed "*"/"+" wildcards to "{name}" / "{wildcard}".
func FiberPathToOpenAPI(path string) string {
	path = nativeParamPattern.ReplaceAllString(path, `{$2}`)
	// ReplaceAll does not re-scan overlapping matches ("/*/*"), so repeat
	// until stable; each pass strictly shrinks the number of bare wildcards.
	for unnamedWildcardPattern.MatchString(path) {
		path = unnamedWildcardPattern.ReplaceAllString(path, `/{wildcard}$1`)
	}
	return path
}

// IrisPathToOpenAPI strips Iris macro types: "{id:int}" becomes "{id}".
func IrisPathToOpenAPI(path string) string {
	return irisParamPattern.ReplaceAllString(path, `{$1}`)
}

// SetOperation stores operation in item's slot for method.
func SetOperation(item *PathItem, method string, operation *Operation) error {
	target, err := item.operationSlot(strings.ToUpper(method))
	if err != nil {
		return fmt.Errorf("%w %q", ErrUnsupportedMethod, method)
	}
	*target = operation
	return nil
}

// operationSlot returns the field holding the operation for an upper-case
// HTTP method, so reads and writes share one method switch.
func (item *PathItem) operationSlot(method string) (**Operation, error) {
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

// pathItemMethods is the fixed, spec-order method list used wherever
// operations are iterated, so output and errors are deterministic.
var pathItemMethods = []string{"GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE"}

// isEmpty reports whether item holds no operation.
func (item *PathItem) isEmpty() bool {
	for _, method := range pathItemMethods {
		if slot, _ := item.operationSlot(method); *slot != nil {
			return false
		}
	}
	return true
}
