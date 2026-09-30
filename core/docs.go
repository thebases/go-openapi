package core

import (
	"net/http"

	openapidocs "github.com/thebases/go-openapi/ui"
)

// Docs exposes the ui package's handlers for manual net/http mounting, so
// callers that only import core can still serve the docs UI and document.
var Docs docsNamespace

type docsNamespace struct{}

// Handler returns the docs UI handler for config; see ui.DocsHandler.
func (docsNamespace) Handler(config DocsConfig) (http.Handler, error) {
	return openapidocs.DocsHandler(config)
}

// DocumentHandler serves api's rendered document with ETag support; see
// ui.DocumentHandler.
func (docsNamespace) DocumentHandler(api *API) http.Handler {
	return openapidocs.DocumentHandler(api)
}
