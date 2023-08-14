package types

import (
	"testing"
)

func TestIsArrayPath(t *testing.T) {
	tests := []struct {
		name          string
		arrayPaths    []string
		path          string
		wantIsArray   bool
		wantArrayPath string
		wantSubPath   string
	}{
		{
			name:          "path matches array path",
			arrayPaths:    []string{"users", "orders"},
			path:          "users",
			wantIsArray:   true,
			wantArrayPath: "users",
			wantSubPath:   "",
		},
		{
			name:          "path is a subpath of array path",
			arrayPaths:    []string{"users", "orders"},
			path:          "users.id",
			wantIsArray:   true,
			wantArrayPath: "users",
			wantSubPath:   "id",
		},
		{
			name:          "path does not match any array path",
			arrayPaths:    []string{"users", "orders"},
			path:          "products",
			wantIsArray:   false,
			wantArrayPath: "",
			wantSubPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SchemaInfo{
				ArrayPaths: tt.arrayPaths,
			}
			gotIsArray, gotArrayPath, gotSubPath := s.IsArrayPath(tt.path)
			if gotIsArray != tt.wantIsArray {
				t.Errorf("Test name: %s - IsArrayPath() gotIsArray = %v, want %v", tt.name, gotIsArray, tt.wantIsArray)
			}
			if gotArrayPath != tt.wantArrayPath {
				t.Errorf("Test name: %s - IsArrayPath() gotArrayPath = %v, want %v", tt.name, gotArrayPath, tt.wantArrayPath)
			}
			if gotSubPath != tt.wantSubPath {
				t.Errorf("Test name: %s - IsArrayPath() gotSubPath = %v, want %v", tt.name, gotSubPath, tt.wantSubPath)
			}
		})
	}
}
