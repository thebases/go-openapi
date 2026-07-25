package openapi

import (
	"fmt"
	"regexp"
	"strings"
)

var fiberParamPattern = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

func FiberPathToOpenAPI(path string) string {
	return fiberParamPattern.ReplaceAllString(path, `{$1}`)
}

func SetOperation(item *PathItem, method string, operation *Operation) error {
	switch strings.ToUpper(method) {
	case "GET":
		item.Get = operation
	case "POST":
		item.Post = operation
	case "PUT":
		item.Put = operation
	case "PATCH":
		item.Patch = operation
	case "DELETE":
		item.Delete = operation
	case "HEAD":
		item.Head = operation
	case "OPTIONS":
		item.Options = operation
	case "TRACE":
		item.Trace = operation
	default:
		return fmt.Errorf("unsupported HTTP method %q", method)
	}

	return nil
}
