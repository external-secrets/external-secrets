package sprig

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

func dfault(d interface{}, given ...interface{}) interface{} {
	if empty(given) || empty(given[0]) {
		return d
	}
	return given[0]
}

func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	switch g.Kind() {
	default:
		return g.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return g.Len() == 0
	case reflect.Bool:
		return !g.Bool()
	case reflect.Complex64, reflect.Complex128:
		return g.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return g.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return g.Float() == 0
	case reflect.Struct:
		return false
	}
}

func coalesce(v ...interface{}) interface{} {
	for _, val := range v {
		if !empty(val) {
			return val
		}
	}
	return nil
}

func all(v ...interface{}) bool {
	for _, val := range v {
		if empty(val) {
			return false
		}
	}
	return true
}

func any(v ...interface{}) bool {
	for _, val := range v {
		if !empty(val) {
			return true
		}
	}
	return false
}

func fromJson(v string) interface{} {
	output, _ := mustFromJson(v)
	return output
}

func mustFromJson(v string) (interface{}, error) {
	var output interface{}
	err := json.Unmarshal([]byte(v), &output)
	return output, err
}

func toJson(v interface{}) string {
	output, _ := json.Marshal(v)
	return string(output)
}

func mustToJson(v interface{}) (string, error) {
	output, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func toPrettyJson(v interface{}) string {
	output, _ := json.MarshalIndent(v, "", "  ")
	return string(output)
}

func mustToPrettyJson(v interface{}) (string, error) {
	output, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func toRawJson(v interface{}) string {
	output, err := mustToRawJson(v)
	if err != nil {
		panic(err)
	}
	return string(output)
}

func mustToRawJson(v interface{}) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(&v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func ternary(vt interface{}, vf interface{}, v bool) interface{} {
	if v {
		return vt
	}

	return vf
}
