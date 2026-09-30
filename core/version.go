package core

// SpecVersion selects which OpenAPI Specification version core.API generates.
// The Schema Object model core builds is JSON Schema 2020-12 shaped (OAS
// 3.1/3.2 semantics); Version30 output is produced by downgrading that model
// at serialization time since OAS 3.0 uses an incompatible Schema Object
// subset (nullable/exclusiveMinimum-as-bool instead of type-array/null).
type SpecVersion int

const (
	Version30 SpecVersion = iota // OpenAPI 3.0.4
	Version31                    // OpenAPI 3.1.1
	Version32                    // OpenAPI 3.2.0

	// DefaultVersion is what core.New produces when WithOpenAPIVersion is not
	// used.
	DefaultVersion = Version32
)

// String returns the OpenAPI version string, such as "3.2.0".
func (v SpecVersion) String() string {
	switch v {
	case Version30:
		return "3.0.4"
	case Version31:
		return "3.1.1"
	case Version32:
		return "3.2.0"
	default:
		return DefaultVersion.String()
	}
}

// WithOpenAPIVersion selects the OpenAPI Specification version core.New
// generates (named WithOpenAPIVersion, not WithVersion, since that name is
// already used by the Info.Version / API-version option):
//
//	0 = OpenAPI 3.0.4
//	1 = OpenAPI 3.1.1
//	2 = OpenAPI 3.2.0 (default)
//
// Untyped constants (WithOpenAPIVersion(0)) and the Version30/31/32 constants
// are both accepted. Any other value makes JSON and Snapshot return
// ErrUnsupportedVersion instead of silently emitting a different version.
func WithOpenAPIVersion(version SpecVersion) Option {
	return func(api *API) {
		api.version = version
		api.doc.OpenAPI = version.String()
	}
}

// valid reports whether v is one of the supported spec versions.
func (v SpecVersion) valid() bool {
	return v >= Version30 && v <= Version32
}
