package core_test

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	core "github.com/thebases/go-openapi/core"
)

func Example() {
	api := core.New(core.WithTitle("Merchant API"), core.WithVersion("1.0.0"))

	err := api.AddOperation(http.MethodGet, "/merchants/{id}", core.Operation{
		OperationID: "getMerchant",
		Parameters:  []core.ParameterOrReference{core.PathParameter("id", core.StringSchema())},
		Responses: map[string]core.ResponseOrReference{
			"200": core.JSONResponse("Merchant found", core.StringSchema()),
		},
	})
	if err != nil {
		panic(err)
	}

	doc, err := api.Snapshot()
	if err != nil {
		panic(err)
	}
	fmt.Println(doc.OpenAPI, doc.Validate() == nil, doc.Paths["/merchants/{id}"].Get.OperationID)
	// Output: 3.2.0 true getMerchant
}

type Merchant struct {
	ID     int64  `json:"id" readOnly:"true"`
	Name   string `json:"name" minLength:"1"`
	Status string `json:"status,omitempty" enum:"active,suspended"`
}

func ExampleReflector() {
	reflector := core.NewReflector()
	ref, err := reflector.ReflectType(reflect.TypeOf(Merchant{}))
	if err != nil {
		panic(err)
	}

	schema := reflector.Components["Merchant"].Value
	fmt.Println(ref.Ref)
	fmt.Println(schema.Required, schema.Properties["id"].Value.Format, schema.Properties["status"].Value.Enum)
	// Output:
	// #/components/schemas/Merchant
	// [id name] int64 [active suspended]
}

// logAdapter is a minimal core.GroupAdapter: it prints instead of mounting
// routes on a real framework.
type logAdapter struct{ prefix string }

func (a logAdapter) Register(method, path string, handlers ...string) error {
	fmt.Println("native:", method, a.prefix+path)
	return nil
}

func (a logAdapter) MountDocs(core.DocsMount) error { return nil }

func (a logAdapter) Group(prefix string) (core.GroupAdapter[string], error) {
	return logAdapter{prefix: a.prefix + prefix}, nil
}

func ExampleNewRootGroup() {
	api := core.New()
	op := core.Operation{Responses: map[string]core.ResponseOrReference{"204": {Value: &core.Response{Description: "Deleted"}}}}

	merchants := core.NewRootGroup[string](api, logAdapter{}).Group("/v1").Group("/merchants")
	if err := merchants.DELETE("/:id", op, "deleteMerchant"); err != nil {
		panic(err)
	}

	for path := range api.Document().Paths {
		fmt.Println("document:", path)
	}
	// Output:
	// native: DELETE /v1/merchants/:id
	// document: /v1/merchants/{id}
}

func ExampleHandle() {
	api := core.New()
	spec := core.Route("/merchants/:id", core.Operation{
		Responses: map[string]core.ResponseOrReference{"200": core.JSONResponse("ok", nil)},
	}).WithMethod(http.MethodGet)

	// A route without handlers is rejected before anything is documented.
	err := core.Handle[string](api, logAdapter{}, spec)
	fmt.Println(errors.Is(err, core.ErrNoHandler), len(api.Document().Paths))
	// Output: true 0
}

func ExampleWithOpenAPIVersion() {
	api := core.New(core.WithOpenAPIVersion(core.Version30))
	raw, err := api.JSON()
	if err != nil {
		panic(err)
	}
	fmt.Println(strings.Contains(string(raw), `"openapi": "3.0.4"`))
	// Output: true
}
