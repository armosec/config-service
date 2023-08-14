package types

import "strings"

var apisInfo = map[string]*APIInfo{}

type APIInfo struct {
	BasePath     string     `json:"basePath"`
	DBCollection string     `json:"dbCollection"`
	Type         string     `json:"type"`
	Schema       SchemaInfo `json:"schema"`
}

type SchemaInfo struct {
	ArrayPaths []string `json:"arrayPaths,omitempty"`
}

func GetAPIInfo(apiName string) *APIInfo {
	if apiInfo, ok := apisInfo[apiName]; ok {
		return apiInfo
	}
	return nil
}

func SetAPIInfo(apiName string, apiInfo *APIInfo) {
	apisInfo[apiName] = apiInfo
}

func (s *SchemaInfo) IsArrayPath(path string) (isArray bool, arrayPath, subPath string) {
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
