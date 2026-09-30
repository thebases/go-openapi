package core

import (
	"encoding/json"
	"errors"
	"reflect"
)

// Reference is the OpenAPI Reference Object ({"$ref": "..."}).
type Reference struct {
	Ref string `json:"$ref"`
}

// SchemaOrReference holds either a $ref (Ref) or an inline Schema (Value); setting both is a marshaling error.
type SchemaOrReference struct {
	Ref   string
	Value *Schema
}

// ParameterOrReference holds either a $ref (Ref) or an inline Parameter (Value).
type ParameterOrReference struct {
	Ref   string
	Value *Parameter
}

// ExampleOrReference holds either a $ref (Ref) or an inline Example (Value).
type ExampleOrReference struct {
	Ref   string
	Value *Example
}

// RequestBodyOrReference holds either a $ref (Ref) or an inline RequestBody (Value).
type RequestBodyOrReference struct {
	Ref   string
	Value *RequestBody
}

// ResponseOrReference holds either a $ref (Ref) or an inline Response (Value).
type ResponseOrReference struct {
	Ref   string
	Value *Response
}

// SecuritySchemeOrReference holds either a $ref (Ref) or an inline SecurityScheme (Value).
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

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v SchemaOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v ParameterOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v ExampleOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v RequestBodyOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v ResponseOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// MarshalJSON emits {"$ref": Ref} or Value, and errors when both are set.
func (v SecuritySchemeOrReference) MarshalJSON() ([]byte, error) {
	return marshalReferenceOrValue(v.Ref, v.Value)
}

// SchemaRef returns a $ref to components.schemas.<name>.
func SchemaRef(name string) *SchemaOrReference {
	return &SchemaOrReference{Ref: "#/components/schemas/" + name}
}

// InlineSchema wraps schema as an inline value.
func InlineSchema(schema *Schema) *SchemaOrReference {
	return &SchemaOrReference{Value: schema}
}

// ExampleRef returns a $ref to components.examples.<name>.
func ExampleRef(name string) *ExampleOrReference {
	return &ExampleOrReference{Ref: "#/components/examples/" + name}
}

// InlineExample wraps example as an inline value.
func InlineExample(example *Example) *ExampleOrReference {
	return &ExampleOrReference{Value: example}
}
