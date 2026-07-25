package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	generatedDocsDir  = "docs"
	generatedJSONPath = "docs/openapi.json"
	generatedYAMLPath = "docs/openapi.yml"
)

func (api *API) renderGeneratedDocuments() ([]byte, error) {
	api.mu.Lock()
	defer api.mu.Unlock()
	return api.renderGeneratedDocumentsLocked()
}

func (api *API) renderGeneratedDocumentsLocked() ([]byte, error) {
	// Always serialize from the in-memory document so the exported artifacts stay
	// aligned with any routes or schemas added after openapi.New() returns.
	if err := resolveDocumentDescriptions(&api.doc); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(api.doc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := writeGeneratedDocuments(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func writeGeneratedDocuments(raw []byte) error {
	if err := os.MkdirAll(filepath.Clean(generatedDocsDir), 0o755); err != nil {
		return fmt.Errorf("openapi: create docs directory: %w", err)
	}
	if err := os.WriteFile(filepath.Clean(generatedJSONPath), raw, 0o644); err != nil {
		return fmt.Errorf("openapi: write generated JSON: %w", err)
	}
	if err := os.WriteFile(filepath.Clean(generatedYAMLPath), yamlFromJSON(raw), 0o644); err != nil {
		return fmt.Errorf("openapi: write generated YAML: %w", err)
	}
	return nil
}

func yamlFromJSON(raw []byte) []byte {
	// YAML 1.2 is a superset of JSON, so the JSON document is a valid .yml export
	// without adding a YAML dependency to the root module.
	return raw
}
