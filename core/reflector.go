package core

import (
	"encoding"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	reflectorTimeType          = reflect.TypeOf(time.Time{})
	reflectorTextUnmarshalerTy = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

type Reflector struct {
	Components map[string]*SchemaOrReference
	Visiting   map[reflect.Type]bool
	Names      map[reflect.Type]string
}

func NewReflector() *Reflector {
	return &Reflector{
		Components: make(map[string]*SchemaOrReference),
		Visiting:   make(map[reflect.Type]bool),
		Names:      make(map[reflect.Type]string),
	}
}

func (r *Reflector) ReflectType(t reflect.Type) (*SchemaOrReference, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot reflect nil type")
	}

	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}

	if schema := specialSchema(t); schema != nil {
		schema.Nullable = nullable
		return InlineSchema(schema), nil
	}

	var ref *SchemaOrReference
	var err error

	switch t.Kind() {
	case reflect.Bool:
		ref = InlineSchema(&Schema{Type: "boolean"})
	case reflect.String:
		ref = InlineSchema(&Schema{Type: "string"})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		ref = InlineSchema(&Schema{Type: "integer", Format: "int32"})
	case reflect.Int64:
		ref = InlineSchema(&Schema{Type: "integer", Format: "int64"})
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		min := float64(0)
		ref = InlineSchema(&Schema{Type: "integer", Format: "int32", Minimum: &min})
	case reflect.Uint64:
		min := float64(0)
		ref = InlineSchema(&Schema{Type: "integer", Format: "int64", Minimum: &min})
	case reflect.Float32:
		ref = InlineSchema(&Schema{Type: "number", Format: "float"})
	case reflect.Float64:
		ref = InlineSchema(&Schema{Type: "number", Format: "double"})
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			ref = InlineSchema(&Schema{Type: "string", Format: "byte"})
		} else {
			var item *SchemaOrReference
			item, err = r.ReflectType(t.Elem())
			if err == nil {
				ref = InlineSchema(&Schema{Type: "array", Items: item})
			}
		}
	case reflect.Array:
		var item *SchemaOrReference
		item, err = r.ReflectType(t.Elem())
		if err == nil {
			length := uint64(t.Len())
			ref = InlineSchema(&Schema{Type: "array", Items: item, MinItems: &length, MaxItems: &length})
		}
	case reflect.Map:
		ref, err = r.reflectMap(t)
	case reflect.Struct:
		ref, err = r.reflectStruct(t)
	case reflect.Interface:
		ref = InlineSchema(&Schema{})
	default:
		err = fmt.Errorf("unsupported Go type %s", t)
	}

	if err != nil {
		return nil, err
	}

	if nullable && ref != nil {
		if ref.Value != nil {
			ref.Value.Nullable = true
		} else if ref.Ref != "" {
			ref = InlineSchema(&Schema{
				Nullable: true,
				AllOf:    []*SchemaOrReference{{Ref: ref.Ref}},
			})
		}
	}

	return ref, nil
}

func specialSchema(t reflect.Type) *Schema {
	if t == reflectorTimeType {
		return &Schema{Type: "string", Format: "date-time"}
	}
	if reflect.PointerTo(t).Implements(reflectorTextUnmarshalerTy) {
		return &Schema{Type: "string"}
	}
	return nil
}

func (r *Reflector) reflectMap(t reflect.Type) (*SchemaOrReference, error) {
	if t.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("OpenAPI object map key must be string, got %s", t.Key())
	}

	valueSchema, err := r.ReflectType(t.Elem())
	if err != nil {
		return nil, err
	}

	return InlineSchema(&Schema{Type: "object", AdditionalProperties: valueSchema}), nil
}

func (r *Reflector) reflectStruct(t reflect.Type) (*SchemaOrReference, error) {
	name := r.schemaName(t)

	if _, exists := r.Components[name]; exists {
		return SchemaRef(name), nil
	}
	if r.Visiting[t] {
		return SchemaRef(name), nil
	}

	r.Visiting[t] = true
	defer delete(r.Visiting, t)

	placeholder := &SchemaOrReference{Value: &Schema{Type: "object", Properties: make(map[string]*SchemaOrReference)}}
	r.Components[name] = placeholder
	schema := placeholder.Value

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		fieldName, options, skip := jsonField(field)
		if skip {
			continue
		}
		if fieldName == "" {
			fieldName = field.Name
		}

		fieldSchema, err := r.ReflectType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("reflect %s.%s: %w", t.Name(), field.Name, err)
		}

		fieldSchema = applyFieldTags(fieldSchema, field)
		schema.Properties[fieldName] = fieldSchema

		if isFieldRequired(field, options) {
			schema.Required = append(schema.Required, fieldName)
		}
	}

	return SchemaRef(name), nil
}

func (r *Reflector) schemaName(t reflect.Type) string {
	if existing := r.Names[t]; existing != "" {
		return existing
	}
	if t.Name() != "" {
		r.Names[t] = t.Name()
		return t.Name()
	}

	name := strings.NewReplacer("*", "", "[]", "Array", "[", "", "]", "", ".", "_").Replace(t.String())
	r.Names[t] = name
	return name
}

func jsonField(field reflect.StructField) (name string, options map[string]bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", nil, true
	}

	options = make(map[string]bool)
	if tag == "" {
		return field.Name, options, false
	}

	parts := strings.Split(tag, ",")
	name = parts[0]
	for _, option := range parts[1:] {
		options[option] = true
	}
	return name, options, false
}

func isFieldRequired(field reflect.StructField, jsonOptions map[string]bool) bool {
	if field.Tag.Get("required") == "true" {
		return true
	}
	if strings.Contains(","+field.Tag.Get("validate")+",", ",required,") {
		return true
	}
	if jsonOptions["omitempty"] {
		return false
	}
	return field.Type.Kind() != reflect.Pointer
}

func applyFieldTags(ref *SchemaOrReference, field reflect.StructField) *SchemaOrReference {
	if ref == nil {
		return nil
	}

	target := ref
	if ref.Ref != "" {
		target = InlineSchema(&Schema{AllOf: []*SchemaOrReference{{Ref: ref.Ref}}})
	}
	if target.Value == nil {
		return target
	}

	schema := target.Value
	schema.Description = field.Tag.Get("description")
	if value := field.Tag.Get("format"); value != "" {
		schema.Format = value
	}
	if value := field.Tag.Get("enum"); value != "" {
		for _, item := range strings.Split(value, ",") {
			schema.Enum = append(schema.Enum, strings.TrimSpace(item))
		}
	}
	if value := field.Tag.Get("example"); value != "" {
		schema.Example = parseLiteral(value, schema.Type)
	}
	if value := field.Tag.Get("default"); value != "" {
		schema.Default = parseLiteral(value, schema.Type)
	}
	if value := field.Tag.Get("minimum"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			schema.Minimum = &parsed
		}
	}
	if value := field.Tag.Get("maximum"); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			schema.Maximum = &parsed
		}
	}
	if value := field.Tag.Get("minLength"); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			schema.MinLength = &parsed
		}
	}
	if value := field.Tag.Get("maxLength"); value != "" {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			schema.MaxLength = &parsed
		}
	}
	if value := field.Tag.Get("pattern"); value != "" {
		schema.Pattern = value
	}
	schema.ReadOnly = field.Tag.Get("readOnly") == "true"
	schema.WriteOnly = field.Tag.Get("writeOnly") == "true"
	schema.Deprecated = field.Tag.Get("deprecated") == "true"
	if field.Type.Kind() == reflect.Pointer || field.Tag.Get("nullable") == "true" {
		schema.Nullable = true
	}

	return target
}

func parseLiteral(value, schemaType string) any {
	switch schemaType {
	case "boolean":
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	case "integer":
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	case "number":
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return value
}
