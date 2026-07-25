package openapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesGeneratedDocsFiles(t *testing.T) {
	restore := withWorkingDirectory(t, t.TempDir())
	defer restore()

	api := New(
		WithTitle("Generated API"),
		WithVersion("1.2.3"),
	)

	jsonRaw, err := os.ReadFile(filepath.Join("docs", "openapi.json"))
	if err != nil {
		t.Fatalf("read generated JSON file after New: %v", err)
	}
	if !strings.Contains(string(jsonRaw), `"Generated API"`) {
		t.Fatalf("expected generated JSON to contain API title, got %s", string(jsonRaw))
	}

	yamlRaw, err := os.ReadFile(filepath.Join("docs", "openapi.yml"))
	if err != nil {
		t.Fatalf("read generated YAML file after New: %v", err)
	}
	if string(yamlRaw) != string(jsonRaw) {
		t.Fatal("generated YAML file does not match returned JSON-compatible YAML")
	}

	raw, err := api.JSON()
	if err != nil {
		t.Fatalf("render JSON: %v", err)
	}
	if string(raw) != string(jsonRaw) {
		t.Fatal("expected JSON() output to match generated file contents")
	}
}

func TestAPIJSONRefreshesGeneratedDocsFiles(t *testing.T) {
	restore := withWorkingDirectory(t, t.TempDir())
	defer restore()

	api := New(
		WithTitle("Original API"),
		WithVersion("1.0.0"),
	)

	api.doc.Info.Title = "Updated API"

	raw, err := api.JSON()
	if err != nil {
		t.Fatalf("render refreshed JSON: %v", err)
	}
	if !strings.Contains(string(raw), `"Updated API"`) {
		t.Fatalf("expected refreshed JSON to contain updated title, got %s", string(raw))
	}

	jsonRaw, err := os.ReadFile(filepath.Join("docs", "openapi.json"))
	if err != nil {
		t.Fatalf("read generated JSON after refresh: %v", err)
	}
	if string(jsonRaw) != string(raw) {
		t.Fatal("expected generated JSON file to be refreshed from current document state")
	}

	yamlRaw, err := os.ReadFile(filepath.Join("docs", "openapi.yml"))
	if err != nil {
		t.Fatalf("read generated YAML after refresh: %v", err)
	}
	if string(yamlRaw) != string(raw) {
		t.Fatal("expected generated YAML file to stay aligned with refreshed JSON")
	}
}
