package utils

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func BoolPointer(b bool) *bool {
	return &b
}

func StringPointer(s string) *string {
	return &s
}

func SameType(a, b interface{}) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

func Interface2String(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		// If the value is an int, convert it to a string
		return strconv.Itoa(v)
	case float64:
		// If the value is a float64, convert it to a string
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		// If the value is a bool, convert it to a string
		return strconv.FormatBool(v)
	case time.Time:
		// If the value is a time.Time, convert it to a string
		return v.Format(time.RFC3339)
	default:
		// If the value is of another type, convert it to a string using fmt.Sprintf
		return fmt.Sprintf("%v", v)
	}
}

func String2Interface(value string) interface{} {
	if i, err := strconv.ParseInt(value, 0, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}
	return value
}

func SplitIgnoreEscaped(s, sep, escape string) []string {
	parts := strings.Split(s, sep)
	var result []string
	var buffer string
	for _, part := range parts {
		if buffer != "" {
			buffer += sep
		}
		buffer += part
		if !strings.HasSuffix(part, escape) {
			result = append(result, buffer)
			buffer = ""
		}
	}
	if buffer != "" {
		result = append(result, buffer)
	}
	return result
}
