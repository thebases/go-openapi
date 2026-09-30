// Package core builds OpenAPI 3.0.4, 3.1.1, and 3.2.0 documents from code.
//
// An API accumulates operations, components, and security schemes; it is
// safe for concurrent use and serves a cached rendering (API.JSON) that is
// rebuilt only after a registration. Reflector turns Go types into schemas.
//
// Framework integrations implement Adapter (and GroupAdapter for nested
// routers); Handle, MountDocs, and NewRootGroup then document and mount routes
// as one unit, so the document only lists routes the framework accepted.
// The integrations live in separate modules under integrations/, keeping this
// module free of third-party dependencies.
package core
