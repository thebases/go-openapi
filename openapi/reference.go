package openapi

import (
	"encoding/json"
	"errors"
	"reflect"
)

type Reference struct {
	Ref string `json:"$ref"`
}

type SchemaOrReference struct {
	Ref   string
	Value *Schema
}

type ParameterOrReference struct {
	Ref   string
	Value *Parameter
}

type ExampleOrReference struct {
	Ref   string
	Value *Example
}

type RequestBodyOrReference struct {
	Ref   string
	Value *RequestBody
}

type ResponseOrReference struct {
	Ref   string
	Value *Response
}

type SecuritySchemeOrReference struct {
	Ref   string
	Value *SecurityScheme
}

func marshalReferenceOrValue(ref string, value any) ([]byte, error) {
	// Values arrive here through interface{} wrappers, so typed nil pointers must
	// be normalized first or plain $ref objects will look like ref+value conflicts.
	hasValue := hasNonNilValue(value)

	switch {
	case ref != "" && hasValue:
		return nil, errors.New("reference and value cannot both be set")
	case ref != "":
		return json.Marshal(Reference{Ref: ref})
	case hasValue:
		return json.Marshal(value)
	default:
		return []byte("null"), nil
	}
}

func hasNonNilValue(value any) bool {
	if value == nil {
		return false
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !reflected.IsNil()
	default:
		return true
	}
}

func (v SchemaOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v ParameterOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v ExampleOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v RequestBodyOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v ResponseOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v SecuritySchemeOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func SchemaRef(name string) *SchemaOrReference {
	return &SchemaOrReference{Ref: "#/components/schemas/" + name}
}

func InlineSchema(schema *Schema) *SchemaOrReference {
	return &SchemaOrReference{Value: schema}
}

func ExampleRef(name string) *ExampleOrReference {
	return &ExampleOrReference{Ref: "#/components/examples/" + name}
}

func InlineExample(example *Example) *ExampleOrReference {
	return &ExampleOrReference{Value: example}
}
