package gocrud

import (
	"reflect"
	"strings"
)

type Reflection struct{}

func (r *Reflection) StructToMap(d interface{}) map[string]any {
	m := make(map[string]interface{})

	val := reflect.ValueOf(d).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		name := field.Tag.Get("db")
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		if name == "reflection" {
			continue
		}
		ptr := val.Field(i).Addr().Interface()
		m[name] = ptr
	}

	return m
}
