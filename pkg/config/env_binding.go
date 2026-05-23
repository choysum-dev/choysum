// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/spf13/viper"
)

func bindConfigEnv(v *viper.Viper) error {
	if v == nil {
		return nil
	}
	return bindStructEnv(v, reflect.TypeOf(Config{}), nil)
}

func bindStructEnv(v *viper.Viper, typ reflect.Type, path []string) error {
	typ = indirectType(typ)
	if typ.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		key := mapstructureFieldKey(field)
		if key == "" || key == "-" {
			continue
		}

		nextPath := append(append([]string(nil), path...), key)
		fieldType := indirectType(field.Type)
		if fieldType.Kind() == reflect.Struct {
			if err := bindStructEnv(v, fieldType, nextPath); err != nil {
				return err
			}
			continue
		}
		if fieldType.Kind() == reflect.Map || fieldType.Kind() == reflect.Interface {
			continue
		}

		if err := v.BindEnv(strings.Join(nextPath, "."), envNameForPath(strings.TrimSpace(v.GetEnvPrefix()), nextPath)); err != nil {
			return err
		}
	}

	return nil
}

func mapstructureFieldKey(field reflect.StructField) string {
	tag := field.Tag.Get("mapstructure")
	if idx := strings.IndexByte(tag, ','); idx >= 0 {
		tag = tag[:idx]
	}
	return strings.TrimSpace(tag)
}

func envNameForPath(prefix string, path []string) string {
	parts := make([]string, 0, len(path)+1)
	if prefix != "" {
		parts = append(parts, strings.ToUpper(prefix))
	}
	for _, segment := range path {
		name := envSegmentName(segment)
		if name == "" {
			continue
		}
		parts = append(parts, name)
	}
	return strings.Join(parts, "_")
}

func envSegmentName(segment string) string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return ""
	}

	runes := []rune(segment)
	var builder strings.Builder
	lastUnderscore := false
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || unicode.IsSpace(r):
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		case unicode.IsUpper(r):
			if shouldInsertEnvUnderscore(runes, i) && builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
			}
			builder.WriteRune(unicode.ToUpper(r))
			lastUnderscore = false
		case unicode.IsLetter(r):
			builder.WriteRune(unicode.ToUpper(r))
			lastUnderscore = false
		case unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		default:
			if builder.Len() > 0 && !lastUnderscore {
				builder.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	return strings.Trim(builder.String(), "_")
}

func shouldInsertEnvUnderscore(runes []rune, index int) bool {
	if index <= 0 || index >= len(runes) {
		return false
	}
	prev := runes[index-1]
	if prev == '_' || prev == '-' || unicode.IsSpace(prev) {
		return false
	}
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}
	if unicode.IsUpper(prev) && index+1 < len(runes) {
		next := runes[index+1]
		return unicode.IsLower(next)
	}
	return false
}

func indirectType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}
