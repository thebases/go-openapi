package core

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

// okOperation is the smallest operation AddOperation accepts.
func okOperation() Operation {
	return Operation{Responses: map[string]ResponseOrReference{"200": JSONResponse("ok", StringSchema())}}
}

// URL deliberately shares its name with net/url.URL (D1).
type URL struct {
	Href string `json:"href"`
}

type Page[T any] struct {
	Items []T `json:"items"`
}

type auditBase struct {
	CreatedAt string `json:"created_at"`
	Name      string `json:"name"`
}

type softDelete struct {
	DeletedAt string `json:"deleted_at"`
}

type embeddingMerchant struct {
	auditBase
	*softDelete
	Name string `json:"name"`
	Kind int    `json:"kind" enum:"1,2"`
}

func TestReflectorSeparatesSameNamedTypesFromDifferentPackages(t *testing.T) {
	r := NewReflector()
	local, err := r.ReflectType(reflect.TypeOf(URL{}))
	if err != nil {
		t.Fatal(err)
	}
	std, err := r.ReflectType(reflect.TypeOf(url.URL{}))
	if err != nil {
		t.Fatal(err)
	}
	if local.Ref == std.Ref {
		t.Fatalf("distinct types share %s", local.Ref)
	}
	if std.Ref != "#/components/schemas/url_URL" {
		t.Fatalf("unexpected qualified name %s", std.Ref)
	}
	// Reflecting again must reuse the names, not mint new ones.
	again, _ := r.ReflectType(reflect.TypeOf(url.URL{}))
	if again.Ref != std.Ref {
		t.Fatalf("name not stable: %s vs %s", again.Ref, std.Ref)
	}
}

func TestReflectorSanitizesGenericNames(t *testing.T) {
	r := NewReflector()
	ref, err := r.ReflectType(reflect.TypeOf(Page[URL]{}))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Ref != "#/components/schemas/Page_URL" {
		t.Fatalf("unexpected generic component ref %s", ref.Ref)
	}
}

func TestReflectorFlattensEmbeddedStructsLikeEncodingJSON(t *testing.T) {
	r := NewReflector()
	if _, err := r.ReflectType(reflect.TypeOf(embeddingMerchant{})); err != nil {
		t.Fatal(err)
	}
	schema := r.Components["embeddingMerchant"].Value
	for _, name := range []string{"created_at", "deleted_at", "name", "kind"} {
		if schema.Properties[name] == nil {
			t.Fatalf("missing flattened property %q in %v", name, schema.Properties)
		}
	}
	if schema.Properties["auditBase"] != nil {
		t.Fatal("embedded struct must not appear as its own property")
	}
	// Fields promoted through a nil-able embedded pointer are optional.
	if strings.Contains(strings.Join(schema.Required, ","), "deleted_at") {
		t.Fatalf("deleted_at must not be required: %v", schema.Required)
	}
	if got := schema.Properties["kind"].Value.Enum; !reflect.DeepEqual(got, []any{int64(1), int64(2)}) {
		t.Fatalf("enum must be typed, got %#v", got)
	}
	if got := schema.Properties["kind"].Value.Format; got != "int64" {
		t.Fatalf("Go int must be int64, got %q", got)
	}
}

func TestReflectorRejectsMalformedTags(t *testing.T) {
	type bad struct {
		Age int `json:"age" minimum:"abc"`
	}
	_, err := NewReflector().ReflectType(reflect.TypeOf(bad{}))
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("expected ErrInvalidTag, got %v", err)
	}
}

func TestReflectorInlinesAnonymousStructs(t *testing.T) {
	r := NewReflector()
	ref, err := r.ReflectType(reflect.TypeOf(struct {
		ID string `json:"id"`
	}{}))
	if err != nil {
		t.Fatal(err)
	}
	if ref.Value == nil || ref.Value.Properties["id"] == nil || len(r.Components) != 0 {
		t.Fatalf("anonymous struct should be inline, got %+v components=%d", ref, len(r.Components))
	}
}

func TestAddOperationUnsupportedMethodLeavesNoPath(t *testing.T) {
	api := New()
	if err := api.AddOperation("FOO", "/x", okOperation()); !errors.Is(err, ErrUnsupportedMethod) {
		t.Fatalf("expected ErrUnsupportedMethod, got %v", err)
	}
	if _, leaked := api.Document().Paths["/x"]; leaked {
		t.Fatal("unsupported method left an empty PathItem")
	}
}

func TestDocumentReturnsDeepCopy(t *testing.T) {
	api := New()
	if err := api.AddOperation(http.MethodGet, "/x", okOperation()); err != nil {
		t.Fatal(err)
	}
	doc := api.Document()
	doc.Paths["/injected"] = &PathItem{}
	doc.Paths["/x"].Get.Summary = "mutated"
	fresh := api.Document()
	if fresh.Paths["/injected"] != nil || fresh.Paths["/x"].Get.Summary != "" {
		t.Fatal("Document() exposed internal state")
	}
}

func TestJSONIsCachedAndRebuiltAfterMutation(t *testing.T) {
	api := New()
	first, err := api.JSON()
	if err != nil {
		t.Fatal(err)
	}
	built := api.built
	if _, err := api.JSON(); err != nil || api.built != built {
		t.Fatal("unchanged document should reuse the cached snapshot")
	}
	first[0] = 'X' // callers own the returned bytes
	if err := api.AddOperation(http.MethodGet, "/x", okOperation()); err != nil {
		t.Fatal(err)
	}
	second, _ := api.JSON()
	if second[0] != '{' || !strings.Contains(string(second), `"/x"`) {
		t.Fatalf("stale or corrupted cache: %s", second)
	}
}

func TestJSONConcurrentWithRegistration(t *testing.T) {
	api := New()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = api.AddOperation(http.MethodGet, "/r"+strings.Repeat("x", i), okOperation())
		}()
		go func() {
			defer wg.Done()
			if _, err := api.JSON(); err != nil {
				t.Error(err)
			}
			_ = api.Document()
		}()
	}
	wg.Wait()
	if got := len(api.Document().Paths); got != 20 {
		t.Fatalf("expected 20 paths, got %d", got)
	}
}

func TestDescriptionFSIsSandboxed(t *testing.T) {
	fsys := fstest.MapFS{"desc/api.md": {Data: []byte("from fs")}}
	api := New(WithDescriptionFS(fsys), WithDescription("desc/api.md"))
	doc, err := api.Snapshot()
	if err != nil || doc.Info.Description != "from fs" {
		t.Fatalf("description not resolved from FS: %q %v", doc.Info.Description, err)
	}
	// The registered model keeps the path; only snapshots hold the content.
	if api.doc.Info.Description != "desc/api.md" {
		t.Fatal("resolution must not mutate the registered document")
	}

	escape := New(WithDescriptionFS(fsys), WithDescription("../secret.md"))
	if _, err := escape.JSON(); !errors.Is(err, ErrInvalidDescription) {
		t.Fatalf("expected ErrInvalidDescription, got %v", err)
	}
}

func TestInvalidSpecVersionFailsLoudly(t *testing.T) {
	if _, err := New(WithOpenAPIVersion(7)).JSON(); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected ErrUnsupportedVersion, got %v", err)
	}
}

// recordingAdapter is a GroupAdapter test double.
type recordingAdapter struct {
	name   string
	log    *[]string
	failOn string
	panics bool
}

func (a *recordingAdapter) Register(method, path string, handlers ...string) error {
	if a.panics {
		panic("native router panic")
	}
	if path == a.failOn {
		return errors.New("native failure")
	}
	*a.log = append(*a.log, a.name+" "+method+" "+path)
	return nil
}

func (a *recordingAdapter) MountDocs(m DocsMount) error {
	*a.log = append(*a.log, a.name+" docs "+m.DocsPath)
	return nil
}

func (a *recordingAdapter) Group(prefix string) (GroupAdapter[string], error) {
	if prefix == "/broken" {
		return nil, errors.New("no group")
	}
	return &recordingAdapter{name: a.name + prefix, log: a.log, failOn: a.failOn}, nil
}

func TestHandleRollsBackWhenNativeRegistrationFails(t *testing.T) {
	api := New()
	var log []string
	adapter := &recordingAdapter{name: "root", log: &log, failOn: "/fail"}
	if err := Handle[string](api, adapter, Route("/fail", okOperation()).WithMethod("GET"), "h"); err == nil {
		t.Fatal("expected native failure")
	}
	if len(api.Document().Paths) != 0 {
		t.Fatal("failed registration left an operation in the document")
	}
	// The rollback frees the slot, so a corrected retry succeeds.
	adapter.failOn = ""
	if err := Handle[string](api, adapter, Route("/fail", okOperation()).WithMethod("GET"), "h"); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
}

func TestHandleRollsBackWhenNativeRouterPanics(t *testing.T) {
	api := New()
	var log []string
	func() {
		defer func() { _ = recover() }()
		_ = Handle[string](api, &recordingAdapter{log: &log, panics: true}, Route("/p", okOperation()).WithMethod("GET"), "h")
	}()
	if len(api.Document().Paths) != 0 {
		t.Fatal("panicking registration left an operation in the document")
	}
}

func TestHandleRequiresAHandler(t *testing.T) {
	var log []string
	err := Handle[string](New(), &recordingAdapter{log: &log}, Route("/x", okOperation()).WithMethod("GET"))
	if !errors.Is(err, ErrNoHandler) {
		t.Fatalf("expected ErrNoHandler, got %v", err)
	}
}

func TestGroupMountsDocsOnlyOnRoot(t *testing.T) {
	api := New(WithDocStyle(DocsSwagger))
	var log []string
	root := NewRootGroup[string](api, &recordingAdapter{name: "root", log: &log})
	deep := root.Group("/a").Group("/b").Group("/c")
	if err := deep.GET("/:id", okOperation(), "h"); err != nil {
		t.Fatal(err)
	}
	if err := root.Group("/other").GET("/", okOperation(), "h"); err != nil {
		t.Fatal(err)
	}
	want := []string{"root docs /docs", "root/a/b/c GET /:id", "root/other GET /"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("got %v want %v", log, want)
	}
	if api.Document().Paths["/a/b/c/{id}"] == nil || api.Document().Paths["/other"] == nil {
		t.Fatalf("unexpected doc paths %v", api.Document().Paths)
	}
}

func TestGroupReportsNativeGroupErrorOnHandle(t *testing.T) {
	var log []string
	group := NewRootGroup[string](New(), &recordingAdapter{log: &log}).Group("/broken").Group("/deeper")
	if group.Err() == nil {
		t.Fatal("expected deferred group error")
	}
	if err := group.GET("/", okOperation(), "h"); err == nil || !strings.Contains(err.Error(), "/broken") {
		t.Fatalf("expected group error from Handle, got %v", err)
	}
}

func TestValidateReportsAllProblemsDeterministically(t *testing.T) {
	doc := Document{
		OpenAPI: "3.1.1",
		Info:    Info{Title: "t", Version: "1"},
		Paths: map[string]*PathItem{
			"/b/{id}": {Get: &Operation{Responses: map[string]ResponseOrReference{"299x": JSONResponse("x", SchemaRef("Missing"))}}},
			"/a": {Get: &Operation{
				Parameters: []ParameterOrReference{QueryParameter("q", nil), QueryParameter("q", nil)},
				Responses:  map[string]ResponseOrReference{"200": JSONResponse("ok", nil)},
			}},
		},
	}
	err := doc.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}
	message := err.Error()
	for _, want := range []string{
		`duplicate query parameter "q"`,
		`invalid response key "299x"`,
		`path template variable {id} has no path parameter`,
		`unresolved $ref "#/components/schemas/Missing"`,
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("missing %q in:\n%s", want, message)
		}
	}
	// /a sorts before /b, so its error must come first.
	if strings.Index(message, `"q"`) > strings.Index(message, "299x") {
		t.Fatalf("errors not in sorted path order:\n%s", message)
	}
	if doc.Validate().Error() != message {
		t.Fatal("validation output is not deterministic")
	}
}

func TestCanonicalPathWildcardsAndConstraints(t *testing.T) {
	cases := map[string]string{
		"/files/*":             "/files/{wildcard}",
		"/files/+":             "/files/{wildcard}",
		"/users/:id?":          "/users/{id}",
		"/users/:id<int>":      "/users/{id}",
		"/users/:id/files/*p":  "/users/{id}/files/{p}",
		"/a/{id:int}/b/{slug}": "/a/{id}/b/{slug}",
		"/*":                   "/{wildcard}",
	}
	for in, want := range cases {
		if got := CanonicalPath(in); got != want {
			t.Errorf("CanonicalPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzCanonicalPathIsIdempotent(f *testing.F) {
	for _, seed := range []string{"/", "/a/:id", "/*", "/x/{id:int}", "/f/+/g", "/:a?/:b<int>"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, path string) {
		once := CanonicalPath(path)
		if twice := CanonicalPath(once); twice != once {
			t.Fatalf("not idempotent: %q -> %q -> %q", path, once, twice)
		}
	})
}

func FuzzJoinPathIsAbsolute(f *testing.F) {
	f.Add("/api", "/x")
	f.Add("", "/")
	f.Add("/api/", "")
	f.Fuzz(func(t *testing.T, prefix, relative string) {
		if got := joinPath(prefix, relative); !strings.HasPrefix(got, "/") {
			t.Fatalf("joinPath(%q, %q) = %q is not absolute", prefix, relative, got)
		}
	})
}

func TestExtensionsAreSerialized(t *testing.T) {
	api := New()
	api.doc.Extensions = map[string]any{"x-logo": map[string]string{"url": "logo.png"}, "x-a": 1}
	op := okOperation()
	op.Extensions = map[string]any{"x-rate-limit": 100}
	if err := api.AddOperation(http.MethodGet, "/x", op); err != nil {
		t.Fatal(err)
	}
	raw, err := api.JSON()
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, want := range []string{`"x-rate-limit": 100`, `"x-logo": {`, `"x-a": 1`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
	if strings.Index(body, `"x-a"`) > strings.Index(body, `"x-logo"`) {
		t.Fatal("extensions must be emitted in sorted order")
	}

	api.doc.Extensions = map[string]any{"bad": true}
	api.rev++
	if _, err := api.JSON(); err == nil || !strings.Contains(err.Error(), "must start with x-") {
		t.Fatalf("expected x- prefix error, got %v", err)
	}
}
