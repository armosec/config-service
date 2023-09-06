package types

import (
	"strings"

	"k8s.io/utils/ptr"
)

var path2apiInfo = map[string]APIInfo{}

type APIInfo struct {
	BasePath     string     `json:"basePath"`
	DBCollection string     `json:"dbCollection"`
	Type         string     `json:"type"`
	Schema       SchemaInfo `json:"schema"`
}

type SchemaInfo struct {
	ArrayPaths []string `json:"arrayPaths,omitempty"`
}

func SetAPIInfo(apiName string, apiInfo APIInfo) {
	path2apiInfo[apiName] = apiInfo
}

func GetAPIInfo(apiName string) *APIInfo {
	if apiInfo, ok := path2apiInfo[apiName]; ok {
		return ptr.To(apiInfo)
	}
	return nil
}

func GetAllPaths() []string {
	paths := make([]string, 0, len(path2apiInfo))
	for path := range path2apiInfo {
		paths = append(paths, path)
	}
	return paths
}

func (s *SchemaInfo) GetArrayDetails(path string) (isArray bool, arrayPath, subPath string) {
	for _, ap := range s.ArrayPaths {
		if strings.HasPrefix(path, ap) {
			isArray = true
			arrayPath = ap
			if strings.HasPrefix(path, arrayPath+".") {
				subPath = strings.TrimPrefix(path, arrayPath+".")
			}
			return
		}
	}
	return
}
