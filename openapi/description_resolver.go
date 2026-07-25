package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func resolveDocumentDescriptions(doc *Document) error {
	if doc == nil {
		return nil
	}
	return resolveDescriptionValue(reflect.ValueOf(doc))
}

func resolveDescriptionValue(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		return resolveDescriptionValue(value.Elem())
	case reflect.Interface:
		if value.IsNil() {
			return nil
		}
		return resolveDescriptionValue(value.Elem())
	case reflect.Struct:
		return resolveDescriptionStruct(value)
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := resolveDescriptionValue(value.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			entry := iter.Value()
			// Map entries are not addressable, so mutable object values need a clone
			// written back after the recursive walk updates nested description fields.
			if !entry.CanAddr() && mutableValueKind(entry.Kind()) {
				clone := reflect.New(entry.Type()).Elem()
				clone.Set(entry)
				if err := resolveDescriptionValue(clone); err != nil {
					return err
				}
				value.SetMapIndex(iter.Key(), clone)
				continue
			}
			if err := resolveDescriptionValue(entry); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveDescriptionStruct(value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := value.Type().Field(i)

		if fieldType.Name == "Description" && fieldValue.Kind() == reflect.String && fieldValue.CanSet() {
			resolved, ok, err := resolveDescriptionPath(fieldValue.String())
			if err != nil {
				return err
			}
			if ok {
				fieldValue.SetString(resolved)
			}
			continue
		}

		if err := resolveDescriptionValue(fieldValue); err != nil {
			return err
		}
	}

	return nil
}

func resolveDescriptionPath(value string) (string, bool, error) {
	candidate := strings.TrimSpace(value)
	if !looksLikeMarkdownPath(candidate) {
		return "", false, nil
	}

	content, err := os.ReadFile(candidate)
	if err != nil {
		return "", false, fmt.Errorf("openapi: read markdown description %q: %w", candidate, err)
	}
	return string(content), true, nil
}

func looksLikeMarkdownPath(value string) bool {
	// Only relative, single-line .md paths are treated as external descriptions
	// so normal prose such as "See docs.md for details" stays untouched.
	return value != "" &&
		!strings.ContainsAny(value, "\r\n") &&
		!strings.Contains(value, " ") &&
		!filepath.IsAbs(value) &&
		filepath.Ext(value) == ".md" &&
		!strings.Contains(value, "://")
}

func mutableValueKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Array, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct:
		return true
	default:
		return false
	}
}
