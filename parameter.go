package openapi

import (
	"fmt"
	"reflect"
	"strings"
)

type Parameter struct {
	Name            string             `json:"name"`
	In              string             `json:"in"`
	Description     string             `json:"description,omitempty"`
	Required        bool               `json:"required,omitempty"`
	Deprecated      bool               `json:"deprecated,omitempty"`
	AllowEmptyValue bool               `json:"allowEmptyValue,omitempty"`
	Style           string             `json:"style,omitempty"`
	Explode         *bool              `json:"explode,omitempty"`
	AllowReserved   bool               `json:"allowReserved,omitempty"`
	Schema          *SchemaOrReference `json:"schema,omitempty"`
	Example         any                `json:"example,omitempty"`
	Extensions      map[string]any     `json:"-"`
}

type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
	Extensions  map[string]any       `json:"-"`
}

type MediaType struct {
	Schema     *SchemaOrReference `json:"schema,omitempty"`
	Example    any                `json:"example,omitempty"`
	Examples   map[string]any     `json:"examples,omitempty"`
	Encoding   map[string]any     `json:"encoding,omitempty"`
	Extensions map[string]any     `json:"-"`
}

type Response struct {
	Description string               `json:"description"`
	Headers     map[string]any       `json:"headers,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Links       map[string]any       `json:"links,omitempty"`
	Extensions  map[string]any       `json:"-"`
}

func (a *API) parametersFor(t reflect.Type) ([]ParameterOrReference, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("parameter input must be struct, got %s", t)
	}

	var result []ParameterOrReference

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

		parameter := &Parameter{
			Name:        name,
			In:          location,
			Description: field.Tag.Get("description"),
			Required: location == "path" ||
				field.Tag.Get("required") == "true" ||
				strings.Contains(","+field.Tag.Get("validate")+",", ",required,"),
			Schema: schema,
		}

		result = append(result, ParameterOrReference{Value: parameter})
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
