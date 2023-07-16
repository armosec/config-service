package utils

import (
	"reflect"
	"strconv"
	"strings"
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
