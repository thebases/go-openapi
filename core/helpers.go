package openapi

func StringSchema() *Schema               { return &Schema{Type: "string"} }
func IntegerSchema(format string) *Schema { return &Schema{Type: "integer", Format: format} }
func ArraySchema(items *Schema) *Schema   { return &Schema{Type: "array", Items: items} }
func RefSchema(name string) *Schema       { return &Schema{Ref: "#/components/schemas/" + name} }

func PathParameter(name string, schema *Schema) Parameter {
	return Parameter{Name: name, In: "path", Required: true, Schema: schema}
}

func QueryParameter(name string, schema *Schema) Parameter {
	return Parameter{Name: name, In: "query", Schema: schema}
}

func JSONResponse(description string, schema *Schema) Response {
	return Response{
		Description: description,
		Content: map[string]MediaType{
			"application/json": {Schema: schema},
		},
	}
}
