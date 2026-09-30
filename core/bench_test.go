package core

import (
	"fmt"
	"net/http"
	"testing"
)

// benchmarkAPI builds a document with n operations.
func benchmarkAPI(b *testing.B, n int) *API {
	b.Helper()
	api := New()
	for i := range n {
		if err := api.AddOperation(http.MethodGet, fmt.Sprintf("/r%d/{id}", i), Operation{
			Parameters: []ParameterOrReference{PathParameter("id", StringSchema())},
			Responses:  map[string]ResponseOrReference{"200": JSONResponse("ok", StringSchema())},
		}); err != nil {
			b.Fatal(err)
		}
	}
	return api
}

// BenchmarkJSONCached is the steady-state /openapi.json cost.
func BenchmarkJSONCached(b *testing.B) {
	api := benchmarkAPI(b, 200)
	if _, err := api.JSON(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := api.JSON(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONRebuild is the cost after every mutation (the old per-request cost).
func BenchmarkJSONRebuild(b *testing.B) {
	api := benchmarkAPI(b, 200)
	b.ReportAllocs()
	for b.Loop() {
		api.mu.Lock()
		api.rev++
		api.mu.Unlock()
		if _, err := api.JSON(); err != nil {
			b.Fatal(err)
		}
	}
}
