package core

import "reflect"

// cloneValue returns a deep copy of v, so stored operations and returned
// snapshots never share maps, slices, or pointers with callers. Structs with
// unexported fields (for example a time.Time inside an Example value) are
// copied by value first, then their exported fields are deep-copied.
// Pointer cycles are not supported; json.Marshal rejects them anyway.
func cloneValue[T any](v T) T {
	src := reflect.ValueOf(&v).Elem()
	dst := reflect.New(src.Type()).Elem()
	copyDeep(dst, src)
	return dst.Interface().(T)
}

// copyDeep copies src into the settable dst, recursing through every
// reference kind. Channels and funcs are shared, as they cannot be copied.
func copyDeep(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Pointer:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.New(src.Type().Elem()))
		copyDeep(dst.Elem(), src.Elem())
	case reflect.Interface:
		if src.IsNil() {
			return
		}
		elem := reflect.New(src.Elem().Type()).Elem()
		copyDeep(elem, src.Elem())
		dst.Set(elem)
	case reflect.Map:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeMapWithSize(src.Type(), src.Len()))
		iter := src.MapRange()
		for iter.Next() {
			value := reflect.New(src.Type().Elem()).Elem()
			copyDeep(value, iter.Value())
			dst.SetMapIndex(iter.Key(), value)
		}
	case reflect.Slice:
		if src.IsNil() {
			return
		}
		dst.Set(reflect.MakeSlice(src.Type(), src.Len(), src.Len()))
		for i := 0; i < src.Len(); i++ {
			copyDeep(dst.Index(i), src.Index(i))
		}
	case reflect.Array:
		for i := 0; i < src.Len(); i++ {
			copyDeep(dst.Index(i), src.Index(i))
		}
	case reflect.Struct:
		dst.Set(src)
		for i := 0; i < src.NumField(); i++ {
			if dst.Field(i).CanSet() {
				copyDeep(dst.Field(i), src.Field(i))
			}
		}
	default:
		dst.Set(src)
	}
}
