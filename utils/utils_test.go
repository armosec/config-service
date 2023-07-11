package utils

import (
	"reflect"
	"testing"
)

func TestSplitIgnoreEscaped(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		sep    string
		escape string
		want   []string
	}{
		{
			name:   "Splitting string with escaped commas",
			s:      "Hello//,World//,How are you?,I am fine//,//,Thank you",
			sep:    ",",
			escape: "//",
			want:   []string{"Hello//,World//,How are you?", "I am fine//,//,Thank you"},
		},
		{
			name:   "Splitting string with escaped pipes",
			s:      "This is a test//|Do not split this//please////don't|But split this",
			sep:    "|",
			escape: "//",
			want:   []string{"This is a test//|Do not split this//please////don't", "But split this"},
		},
		{
			name:   "simple split",
			s:      "one|two|three",
			sep:    "|",
			escape: "//",
			want:   []string{"one", "two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitIgnoreEscaped(tt.s, tt.sep, tt.escape); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitIgnoreEscaped() = %v, want %v", got, tt.want)
			}
		})
	}
}
