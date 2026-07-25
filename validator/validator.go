package validator

import spec "github.com/thebases/go-openapi/openapi"

func ValidateDocument(d *spec.Document) error {
	return d.Validate()
}
