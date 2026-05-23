// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package snapshot

import "reflect"

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = cloneAnyValue(value)
	}
	return cloned
}

func cloneAnyValue(value any) any {
	if value == nil {
		return nil
	}
	cloned := cloneAnyReflectValue(reflect.ValueOf(value))
	if !cloned.IsValid() {
		return nil
	}
	return cloned.Interface()
}

func cloneAnyReflectValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return reflect.Value{}
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneAnyReflectValue(value.Elem())
		out := reflect.New(value.Type()).Elem()
		if cloned.IsValid() {
			out.Set(cloned)
		}
		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			cloned.SetMapIndex(iter.Key(), cloneAnyReflectValue(iter.Value()))
		}
		return cloned
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for idx := 0; idx < value.Len(); idx++ {
			cloned.Index(idx).Set(cloneAnyReflectValue(value.Index(idx)))
		}
		return cloned
	default:
		return value
	}
}
