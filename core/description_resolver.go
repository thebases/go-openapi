package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// descriptionResolver replaces "*.md" Description values with file content
// read from fsys (os.DirFS(".") when nil). It walks the document with
// reflection so every Description field is covered without a per-type list.
type descriptionResolver struct {
	fsys fs.FS
}

func (r descriptionResolver) resolveDocument(doc *Document) error {
	if doc == nil {
		return nil
	}
	if r.fsys == nil {
		r.fsys = os.DirFS(".")
	}
	return r.resolveValue(reflect.ValueOf(doc))
}

func (r descriptionResolver) resolveValue(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Pointer, reflect.Interface:
		if value.IsNil() {
			return nil
		}
		return r.resolveValue(value.Elem())
	case reflect.Struct:
		return r.resolveStruct(value)
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := r.resolveValue(value.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		return r.resolveMap(value)
	}

	return nil
}

func (r descriptionResolver) resolveMap(value reflect.Value) error {
	iter := value.MapRange()
	for iter.Next() {
		entry := iter.Value()
		// Map entries are not addressable, so mutable object values need a clone
		// written back after the recursive walk updates nested description fields.
		if !entry.CanAddr() && mutableValueKind(entry.Kind()) {
			clone := reflect.New(entry.Type()).Elem()
			clone.Set(entry)
			if err := r.resolveValue(clone); err != nil {
				return err
			}
			value.SetMapIndex(iter.Key(), clone)
			continue
		}
		if err := r.resolveValue(entry); err != nil {
			return err
		}
	}
	return nil
}

func (r descriptionResolver) resolveStruct(value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldType := value.Type().Field(i)

		if fieldType.Name == "Description" && fieldValue.Kind() == reflect.String && fieldValue.CanSet() {
			resolved, ok, err := r.resolvePath(fieldValue.String())
			if err != nil {
				return err
			}
			if ok {
				fieldValue.SetString(resolved)
			}
			continue
		}

		if err := r.resolveValue(fieldValue); err != nil {
			return err
		}
	}

	return nil
}

// resolvePath reads value from the FS when it looks like a Markdown path.
// fs.ValidPath rejects ".." and rooted paths, so a description can never
// read outside the configured filesystem.
func (r descriptionResolver) resolvePath(value string) (string, bool, error) {
	candidate := strings.TrimSpace(value)
	if !looksLikeMarkdownPath(candidate) {
		return "", false, nil
	}

	name := strings.TrimPrefix(filepath.ToSlash(candidate), "./")
	if !fs.ValidPath(name) {
		return "", false, fmt.Errorf("%w %q: must be relative without \"..\"", ErrInvalidDescription, candidate)
	}

	content, err := fs.ReadFile(r.fsys, name)
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
