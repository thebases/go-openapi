package openapi

import (
	"encoding/json"
	"errors"
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
	switch {
	case ref != "" && value != nil:
		return nil, errors.New("reference and value cannot both be set")
	case ref != "":
		return json.Marshal(Reference{Ref: ref})
	case value != nil:
		return json.Marshal(value)
	default:
		return []byte("null"), nil
	}
}

func (v SchemaOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

func (v ParameterOrReference) MarshalJSON() ([]byte, error) {
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
