package helper

import (
	"encoding/json"
	"log"
	"reflect"
	"strings"
)

// SetFieldByTag sets a struct field by matching a JSON tag, supporting both string and []string types.
func SetFieldByTag(obj interface{}, tag string, value string) {
	val := reflect.ValueOf(obj).Elem()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Match struct field's tag with the given key
		if fieldType.Tag.Get("json") == tag && field.CanSet() {
			switch field.Kind() {
			case reflect.Slice:
				// Try parsing JSON array
				var strSlice []string
				if err := json.Unmarshal([]byte(value), &strSlice); err == nil {
					field.Set(reflect.ValueOf(strSlice))
				} else {
					// If not JSON, assume comma-separated values
					field.Set(reflect.ValueOf(strings.Split(value, ",")))
				}
			case reflect.String:
				// Directly set string values
				field.SetString(value)
			default:
				log.Printf("Unsupported field type for tag %s", tag)
			}
		}
	}
}
