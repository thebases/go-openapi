package core

import (
	"encoding"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reflectorTimeType          = reflect.TypeOf(time.Time{})
	reflectorTextUnmarshalerTy = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

	// packageQualifierPattern matches "github.com/org/pkg." qualifiers inside
	// generic type names such as "Page[github.com/org/pkg.User]".
	packageQualifierPattern = regexp.MustCompile(`(?:[\w.\-]+/)*[\w\-]+\.`)
	// invalidSchemaNameChars matches everything OAS forbids in component keys.
	invalidSchemaNameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
)

// SchemaNamer chooses the base component name for a named Go type. The
// Reflector still sanitizes the result to the OAS component-key alphabet and
// disambiguates collisions between distinct types.
type SchemaNamer interface {
	SchemaName(t reflect.Type) string
}

// RequiredPolicy decides whether a struct field is listed in "required".
// omitEmpty is true for `json:",omitempty"` and `json:",omitzero"`.
type RequiredPolicy interface {
	IsRequired(field reflect.StructField, omitEmpty bool) bool
}

// DefaultSchemaNamer names a type after its Go name with package qualifiers
// removed from generic arguments ("Page[pkg.User]" becomes "Page[User]").
type DefaultSchemaNamer struct{}

// SchemaName implements SchemaNamer.
func (DefaultSchemaNamer) SchemaName(t reflect.Type) string {
	return packageQualifierPattern.ReplaceAllString(t.Name(), "")
}

// DefaultRequiredPolicy marks a field required when it has `required:"true"`
// or `validate:"required"`, or when it is a non-pointer without omitempty.
type DefaultRequiredPolicy struct{}

// IsRequired implements RequiredPolicy.
func (DefaultRequiredPolicy) IsRequired(field reflect.StructField, omitEmpty bool) bool {
	if field.Tag.Get("required") == "true" {
		return true
	}
	if strings.Contains(","+field.Tag.Get("validate")+",", ",required,") {
		return true
	}
	if omitEmpty {
		return false
	}
	return field.Type.Kind() != reflect.Pointer
}

// Reflector converts Go types into OpenAPI schemas. Named struct types become
// reusable components in Components and are referenced by $ref. A Reflector
// is not safe for concurrent use.
type Reflector struct {
	// Components holds every reflected named struct schema by component name.
	Components map[string]*SchemaOrReference
	// Names maps each reflected type to its component name.
	Names map[reflect.Type]string

	// Namer overrides DefaultSchemaNamer.
	Namer SchemaNamer
	// Required overrides DefaultRequiredPolicy.
	Required RequiredPolicy

	// owners maps a component name back to the type that claimed it, so two
	// distinct types never share one component.
	owners map[string]reflect.Type
}

// NewReflector returns an empty Reflector with the default naming and
// required-field policies.
func NewReflector() *Reflector {
	return &Reflector{
		Components: make(map[string]*SchemaOrReference),
		Names:      make(map[reflect.Type]string),
		owners:     make(map[string]reflect.Type),
	}
}

// init lets a zero-value &Reflector{} work like NewReflector().
func (r *Reflector) init() {
	if r.Components == nil {
		r.Components = make(map[string]*SchemaOrReference)
	}
	if r.Names == nil {
		r.Names = make(map[reflect.Type]string)
	}
	if r.owners == nil {
		r.owners = make(map[string]reflect.Type, len(r.Names))
		for t, name := range r.Names {
			r.owners[name] = t
		}
	}
}

// ReflectType returns the schema for t. Pointers become nullable; named
// structs are stored in Components and returned as a $ref.
func (r *Reflector) ReflectType(t reflect.Type) (*SchemaOrReference, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot reflect nil type")
	}
	r.init()

	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}

	ref, err := r.reflectNonPointer(t)
	if err != nil {
		return nil, err
	}
	if !nullable {
		return ref, nil
	}

	if ref.Value != nil {
		ref.Value.Type = nullableType(ref.Value.Type)
		return ref, nil
	}
	// allOf cannot express "ref OR null" since both branches must hold
	// simultaneously, so a nullable $ref uses oneOf with an explicit null type.
	return InlineSchema(&Schema{
		OneOf: []*SchemaOrReference{{Ref: ref.Ref}, {Value: &Schema{Type: "null"}}},
	}), nil
}

// reflectNonPointer maps a non-pointer Go type to a schema.
func (r *Reflector) reflectNonPointer(t reflect.Type) (*SchemaOrReference, error) {
	if schema := specialSchema(t); schema != nil {
		return InlineSchema(schema), nil
	}
	if schema := scalarSchema(t.Kind()); schema != nil {
		return InlineSchema(schema), nil
	}

	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return InlineSchema(&Schema{Type: "string", Format: "byte"}), nil
		}
		item, err := r.ReflectType(t.Elem())
		if err != nil {
			return nil, err
		}
		return InlineSchema(&Schema{Type: "array", Items: item}), nil
	case reflect.Array:
		item, err := r.ReflectType(t.Elem())
		if err != nil {
			return nil, err
		}
		length := uint64(t.Len())
		return InlineSchema(&Schema{Type: "array", Items: item, MinItems: &length, MaxItems: &length}), nil
	case reflect.Map:
		return r.reflectMap(t)
	case reflect.Struct:
		return r.reflectStruct(t)
	case reflect.Interface:
		return InlineSchema(&Schema{}), nil
	default:
		return nil, fmt.Errorf("unsupported Go type %s", t)
	}
}

// scalarSchema maps bool, integer, float, and string kinds. Go int and uint
// are 64-bit on every supported platform, so they must not be int32; unsigned
// kinds that do not fit the signed format get only a minimum of 0.
func scalarSchema(kind reflect.Kind) *Schema {
	zero := float64(0)
	switch kind {
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Int, reflect.Int64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Uint8, reflect.Uint16:
		return &Schema{Type: "integer", Format: "int32", Minimum: &zero}
	case reflect.Uint32:
		return &Schema{Type: "integer", Format: "int64", Minimum: &zero}
	case reflect.Uint, reflect.Uint64, reflect.Uintptr:
		return &Schema{Type: "integer", Minimum: &zero}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	default:
		return nil
	}
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

// reflectStruct stores a named struct as a component and returns its $ref.
// Anonymous structs cannot recurse and have no stable name, so they are
// inlined instead.
func (r *Reflector) reflectStruct(t reflect.Type) (*SchemaOrReference, error) {
	if t.Name() == "" {
		schema := &Schema{Type: "object", Properties: make(map[string]*SchemaOrReference)}
		if err := r.fillStruct(schema, t); err != nil {
			return nil, err
		}
		return InlineSchema(schema), nil
	}

	name := r.schemaName(t)
	if _, exists := r.Components[name]; exists {
		return SchemaRef(name), nil
	}

	// The placeholder is visible before the fields are reflected, so a
	// recursive field resolves to a $ref instead of recursing forever.
	schema := &Schema{Type: "object", Properties: make(map[string]*SchemaOrReference)}
	r.Components[name] = InlineSchema(schema)
	if err := r.fillStruct(schema, t); err != nil {
		delete(r.Components, name)
		return nil, err
	}
	return SchemaRef(name), nil
}

// fillStruct adds t's JSON-visible fields to schema.
func (r *Reflector) fillStruct(schema *Schema, t reflect.Type) error {
	required := r.Required
	if required == nil {
		required = DefaultRequiredPolicy{}
	}

	for _, field := range jsonFields(t) {
		fieldSchema, err := r.ReflectType(field.field.Type)
		if err != nil {
			return fmt.Errorf("reflect %s.%s: %w", t.Name(), field.field.Name, err)
		}
		fieldSchema, err = applyFieldTags(fieldSchema, field.field)
		if err != nil {
			return fmt.Errorf("reflect %s.%s: %w", t.Name(), field.field.Name, err)
		}
		schema.Properties[field.name] = fieldSchema

		// Fields promoted through an embedded pointer vanish when it is nil.
		if !field.viaPointer && required.IsRequired(field.field, field.omitEmpty) {
			schema.Required = append(schema.Required, field.name)
		}
	}
	return nil
}

// schemaName returns the component name for t, unique across distinct types:
// a clash gets the package name prepended ("b_User"), then a numeric suffix.
func (r *Reflector) schemaName(t reflect.Type) string {
	if existing, ok := r.Names[t]; ok {
		return existing
	}

	namer := r.Namer
	if namer == nil {
		namer = DefaultSchemaNamer{}
	}
	base := sanitizeSchemaName(namer.SchemaName(t))

	name := base
	if r.nameTaken(name, t) {
		name = sanitizeSchemaName(path.Base(t.PkgPath()) + "_" + base)
	}
	for suffix := 2; r.nameTaken(name, t); suffix++ {
		name = fmt.Sprintf("%s_%d", base, suffix)
	}

	r.owners[name] = t
	r.Names[t] = name
	return name
}

// nameTaken reports whether name is claimed by a type other than t, including
// components added to Components by hand.
func (r *Reflector) nameTaken(name string, t reflect.Type) bool {
	if owner, ok := r.owners[name]; ok {
		return owner != t
	}
	_, exists := r.Components[name]
	return exists
}

// sanitizeSchemaName maps a type name onto ^[a-zA-Z0-9._-]+$:
// "Page[User,Tag]" becomes "Page_User_Tag".
func sanitizeSchemaName(name string) string {
	name = strings.NewReplacer("*", "", "]", "", " ", "").Replace(name)
	name = invalidSchemaNameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		return "Schema"
	}
	return name
}

// structField is one JSON-visible field of a struct, possibly promoted from
// an embedded struct.
type structField struct {
	name       string
	field      reflect.StructField
	depth      int
	tagged     bool
	omitEmpty  bool
	viaPointer bool
}

// jsonFields lists the fields encoding/json would marshal for t: untagged
// embedded structs are flattened, and name conflicts are resolved by the
// shallowest (then the only tagged) field, or dropped when ambiguous.
func jsonFields(t reflect.Type) []structField {
	var fields []structField
	collectFields(&fields, t, 0, false, map[reflect.Type]bool{})
	return dominantFields(fields)
}

// collectFields appends t's fields to out, descending into embedded structs.
// visiting guards the current embedding path against pointer cycles.
func collectFields(out *[]structField, t reflect.Type, depth int, viaPointer bool, visiting map[reflect.Type]bool) {
	if visiting[t] {
		return
	}
	visiting[t] = true
	defer delete(visiting, t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tagName, options, skip := jsonTag(field)
		if skip {
			continue
		}

		if field.Anonymous && tagName == "" {
			embedded, isPointer := field.Type, false
			if embedded.Kind() == reflect.Pointer {
				embedded, isPointer = embedded.Elem(), true
			}
			if embedded.Kind() == reflect.Struct && specialSchema(embedded) == nil {
				collectFields(out, embedded, depth+1, viaPointer || isPointer, visiting)
				continue
			}
		}
		if !field.IsExported() {
			continue
		}

		name := tagName
		if name == "" {
			name = field.Name
		}
		*out = append(*out, structField{
			name:       name,
			field:      field,
			depth:      depth,
			tagged:     tagName != "",
			omitEmpty:  options["omitempty"] || options["omitzero"],
			viaPointer: viaPointer,
		})
	}
}

// dominantFields keeps one field per JSON name, in first-seen order.
func dominantFields(fields []structField) []structField {
	byName := make(map[string][]structField, len(fields))
	order := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, seen := byName[field.name]; !seen {
			order = append(order, field.name)
		}
		byName[field.name] = append(byName[field.name], field)
	}

	kept := make([]structField, 0, len(order))
	for _, name := range order {
		if field, ok := dominantField(byName[name]); ok {
			kept = append(kept, field)
		}
	}
	return kept
}

// dominantField applies encoding/json's conflict rule to fields sharing a name.
func dominantField(candidates []structField) (structField, bool) {
	minDepth := candidates[0].depth
	for _, candidate := range candidates[1:] {
		minDepth = min(minDepth, candidate.depth)
	}

	var shallow, tagged []structField
	for _, candidate := range candidates {
		if candidate.depth != minDepth {
			continue
		}
		shallow = append(shallow, candidate)
		if candidate.tagged {
			tagged = append(tagged, candidate)
		}
	}

	switch {
	case len(shallow) == 1:
		return shallow[0], true
	case len(tagged) == 1:
		return tagged[0], true
	default:
		return structField{}, false
	}
}

// jsonTag parses the json tag: the explicit name ("" when absent), its
// options, and whether the field is skipped with `json:"-"`.
func jsonTag(field reflect.StructField) (name string, options map[string]bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", nil, true
	}

	options = make(map[string]bool)
	parts := strings.Split(tag, ",")
	for _, option := range parts[1:] {
		options[option] = true
	}
	return parts[0], options, false
}

// applyFieldTags applies the documentation struct tags to a field schema. A
// $ref schema is wrapped in allOf first, since siblings of $ref are unreliable.
func applyFieldTags(ref *SchemaOrReference, field reflect.StructField) (*SchemaOrReference, error) {
	target := ref
	if ref.Ref != "" {
		target = InlineSchema(&Schema{AllOf: []*SchemaOrReference{{Ref: ref.Ref}}})
	}
	if target.Value == nil {
		return target, nil
	}

	schema := target.Value
	if err := applyStringTags(schema, field); err != nil {
		return nil, err
	}
	if err := applyNumericTags(schema, field); err != nil {
		return nil, err
	}

	schema.ReadOnly = field.Tag.Get("readOnly") == "true"
	schema.WriteOnly = field.Tag.Get("writeOnly") == "true"
	schema.Deprecated = field.Tag.Get("deprecated") == "true"
	if field.Type.Kind() == reflect.Pointer || field.Tag.Get("nullable") == "true" {
		markNullable(schema)
	}
	return target, nil
}

// applyStringTags applies description, format, pattern, enum, example, and
// default; the last three are parsed as the schema's primary type.
func applyStringTags(schema *Schema, field reflect.StructField) error {
	if value := field.Tag.Get("description"); value != "" {
		schema.Description = value
	}
	if value := field.Tag.Get("format"); value != "" {
		schema.Format = value
	}
	if value := field.Tag.Get("pattern"); value != "" {
		schema.Pattern = value
	}

	typeName := primaryTypeName(schema.Type)
	if value := field.Tag.Get("enum"); value != "" {
		for _, item := range strings.Split(value, ",") {
			parsed, err := parseLiteral(strings.TrimSpace(item), typeName)
			if err != nil {
				return fmt.Errorf("%w: enum %q: %v", ErrInvalidTag, item, err)
			}
			schema.Enum = append(schema.Enum, parsed)
		}
	}
	for _, tag := range []string{"example", "default"} {
		value := field.Tag.Get(tag)
		if value == "" {
			continue
		}
		parsed, err := parseLiteral(value, typeName)
		if err != nil {
			return fmt.Errorf("%w: %s %q: %v", ErrInvalidTag, tag, value, err)
		}
		if tag == "example" {
			schema.Example = parsed
		} else {
			schema.Default = parsed
		}
	}
	return nil
}

// applyNumericTags applies minimum/maximum and minLength/maxLength, rejecting
// malformed values instead of silently dropping the constraint.
func applyNumericTags(schema *Schema, field reflect.StructField) error {
	for tag, target := range map[string]**float64{"minimum": &schema.Minimum, "maximum": &schema.Maximum} {
		value := field.Tag.Get(tag)
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%w: %s %q", ErrInvalidTag, tag, value)
		}
		*target = &parsed
	}
	for tag, target := range map[string]**uint64{"minLength": &schema.MinLength, "maxLength": &schema.MaxLength} {
		value := field.Tag.Get(tag)
		if value == "" {
			continue
		}
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s %q", ErrInvalidTag, tag, value)
		}
		*target = &parsed
	}
	return nil
}

// markNullable adds null to schema. The allOf $ref wrapper built by
// applyFieldTags cannot express nullability, so it switches to oneOf with an
// explicit null branch.
func markNullable(schema *Schema) {
	if len(schema.AllOf) == 1 && schema.AllOf[0].Ref != "" && schema.Type == nil {
		schema.OneOf = []*SchemaOrReference{schema.AllOf[0], {Value: &Schema{Type: "null"}}}
		schema.AllOf = nil
		return
	}
	schema.Type = nullableType(schema.Type)
}

// parseLiteral converts a tag literal to the JSON type of the schema; other
// types (string, array, ...) keep the raw string.
func parseLiteral(value, schemaType string) (any, error) {
	switch schemaType {
	case "boolean":
		return strconv.ParseBool(value)
	case "integer":
		return strconv.ParseInt(value, 10, 64)
	case "number":
		return strconv.ParseFloat(value, 64)
	default:
		return value, nil
	}
}
