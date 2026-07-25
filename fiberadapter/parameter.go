package fiberadapter

import (
	"fmt"
	"reflect"
	"strings"

	spec "github.com/thebases/go-openapi/openapi"
)

func (a *API) parametersFor(t reflect.Type) ([]spec.ParameterOrReference, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("parameter input must be struct, got %s", t)
	}

	var result []spec.ParameterOrReference

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		location, name := parameterTag(field)
		if location == "" {
			continue
		}

		schema, err := a.reflector.ReflectType(field.Type)
		if err != nil {
			return nil, err
		}

		parameter := &spec.Parameter{
			Name:        name,
			In:          location,
			Description: field.Tag.Get("description"),
			Required: location == "path" ||
				field.Tag.Get("required") == "true" ||
				strings.Contains(","+field.Tag.Get("validate")+",", ",required,"),
			Schema: schema,
		}

		result = append(result, spec.ParameterOrReference{Value: parameter})
	}

	return result, nil
}

func parameterTag(field reflect.StructField) (location, name string) {
	if name := field.Tag.Get("path"); name != "" {
		return "path", name
	}
	if name := field.Tag.Get("query"); name != "" {
		return "query", name
	}
	if name := field.Tag.Get("header"); name != "" {
		return "header", name
	}
	if name := field.Tag.Get("cookie"); name != "" {
		return "cookie", name
	}
	return "", ""
}
