package openapi

import (
	"regexp"
	"strings"
)

var nativeParamPattern = regexp.MustCompile(`(^|/)([:*])([A-Za-z_][A-Za-z0-9_]*)`)

// CanonicalPath converts common router parameter formats to OpenAPI syntax.
func CanonicalPath(path string) string {
	return nativeParamPattern.ReplaceAllStringFunc(path, func(match string) string {
		prefix := ""
		token := match
		if strings.HasPrefix(match, "/") {
			prefix = "/"
			token = strings.TrimPrefix(match, "/")
		}

		return prefix + "{" + token[1:] + "}"
	})
}
