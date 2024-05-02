package types

import (
	"strings"

	"k8s.io/utils/ptr"
)

var path2apiInfo = map[string]APIInfo{}

type APIInfo struct {
	BasePath     string     `json:"basePath"`
	DBCollection string     `json:"dbCollection"`
	Schema       SchemaInfo `json:"schema"`
}

type FieldType string

const (
	Date FieldType = "date"
	//Explict string type in schema is needed to avoid type conversion if this field is a string but holds numbers or bool values
	String FieldType = "string"
)

type SchemaInfo struct {
	ArrayPaths         []string             `json:"arrayPaths,omitempty"`
	FieldsType         map[string]FieldType `json:"fieldsType,omitempty"`
	TimestampFieldName *string              `json:"timestampFieldName,omitempty"` // pointer so empty string can be distinguished from nil
	MustExcludeFields  []string             `json:"mustExcludeFields,omitempty"`  // fields that must be excluded from the response
}

func SetAPIInfo(path string, apiInfo APIInfo) {
	path2apiInfo[path] = apiInfo
}

func GetAPIInfo(path string) *APIInfo {
	if apiInfo, ok := path2apiInfo[path]; ok {
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

func (s *SchemaInfo) IsDate(field string) bool {
	fieldType, exist := s.FieldsType[field]
	return exist && fieldType == Date
}

func (s *SchemaInfo) IsString(field string) bool {
	fieldType, exist := s.FieldsType[field]
	return exist && fieldType == String
}

func (s SchemaInfo) GetTimestampFieldName() string {
	if s.TimestampFieldName == nil {
		return "creationTime"
	}
	return *s.TimestampFieldName
}

func (s SchemaInfo) GetMustExcludeFields() []string {
	return s.MustExcludeFields
}
