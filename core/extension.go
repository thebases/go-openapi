package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// marshalWithExtensions marshals base (a type without MarshalJSON, so this
// does not recurse) and splices the "x-" extensions into the resulting
// object in sorted key order. Objects without extensions cost one plain
// Marshal; there is no unmarshal/re-marshal round trip.
func marshalWithExtensions(base any, extensions map[string]any) ([]byte, error) {
	raw, err := json.Marshal(base)
	if err != nil || len(extensions) == 0 {
		return raw, err
	}

	keys := make([]string, 0, len(extensions))
	for key := range extensions {
		if !strings.HasPrefix(key, "x-") {
			return nil, fmt.Errorf("openapi: extension %q must start with x-", key)
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	// raw is a JSON object: reopen it by dropping the closing brace.
	var out bytes.Buffer
	out.Grow(len(raw) + 32*len(keys))
	out.Write(raw[:len(raw)-1])
	for i, key := range keys {
		value, err := json.Marshal(extensions[key])
		if err != nil {
			return nil, fmt.Errorf("openapi: extension %q: %w", key, err)
		}
		// "{}" has no members yet, so the first extension needs no comma.
		if i > 0 || len(raw) > 2 {
			out.WriteByte(',')
		}
		name, _ := json.Marshal(key)
		out.Write(name)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// Each MarshalJSON converts to a local type without methods, so json.Marshal
// encodes the declared fields and marshalWithExtensions adds Extensions
// (which every struct tags `json:"-"`).

// MarshalJSON emits the Document fields plus its x- Extensions.
func (v Document) MarshalJSON() ([]byte, error) {
	type plain Document
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Info fields plus its x- Extensions.
func (v Info) MarshalJSON() ([]byte, error) {
	type plain Info
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Contact fields plus its x- Extensions.
func (v Contact) MarshalJSON() ([]byte, error) {
	type plain Contact
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the License fields plus its x- Extensions.
func (v License) MarshalJSON() ([]byte, error) {
	type plain License
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Server fields plus its x- Extensions.
func (v Server) MarshalJSON() ([]byte, error) {
	type plain Server
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the ServerVariable fields plus its x- Extensions.
func (v ServerVariable) MarshalJSON() ([]byte, error) {
	type plain ServerVariable
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the ExternalDocumentation fields plus its x- Extensions.
func (v ExternalDocumentation) MarshalJSON() ([]byte, error) {
	type plain ExternalDocumentation
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Tag fields plus its x- Extensions.
func (v Tag) MarshalJSON() ([]byte, error) {
	type plain Tag
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Components fields plus its x- Extensions.
func (v Components) MarshalJSON() ([]byte, error) {
	type plain Components
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the PathItem fields plus its x- Extensions.
func (v PathItem) MarshalJSON() ([]byte, error) {
	type plain PathItem
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Operation fields plus its x- Extensions.
func (v Operation) MarshalJSON() ([]byte, error) {
	type plain Operation
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Parameter fields plus its x- Extensions.
func (v Parameter) MarshalJSON() ([]byte, error) {
	type plain Parameter
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the RequestBody fields plus its x- Extensions.
func (v RequestBody) MarshalJSON() ([]byte, error) {
	type plain RequestBody
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the MediaType fields plus its x- Extensions.
func (v MediaType) MarshalJSON() ([]byte, error) {
	type plain MediaType
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Response fields plus its x- Extensions.
func (v Response) MarshalJSON() ([]byte, error) {
	type plain Response
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Schema fields plus its x- Extensions.
func (v Schema) MarshalJSON() ([]byte, error) {
	type plain Schema
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the SecurityScheme fields plus its x- Extensions.
func (v SecurityScheme) MarshalJSON() ([]byte, error) {
	type plain SecurityScheme
	return marshalWithExtensions(plain(v), v.Extensions)
}

// MarshalJSON emits the Example fields plus its x- Extensions.
func (v Example) MarshalJSON() ([]byte, error) {
	type plain Example
	return marshalWithExtensions(plain(v), v.Extensions)
}
