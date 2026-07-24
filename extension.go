package openapi

import (
	"encoding/json"
	"fmt"
	"strings"
)

func marshalWithExtensions(base any, extensions map[string]any) ([]byte, error) {
	raw, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}

	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}

	for key, value := range extensions {
		if !strings.HasPrefix(key, "x-") {
			return nil, fmt.Errorf("extension %q must start with x-", key)
		}
		values[key] = value
	}

	return json.Marshal(values)
}

// Add custom MarshalJSON methods for each extensible object as the library matures.
// Use alias types inside each method to avoid infinite recursion.
