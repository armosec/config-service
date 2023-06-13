package db

import (
	"config-service/types"
	"strings"

	"github.com/chidiwilliams/flatbson"

	"go.mongodb.org/mongo-driver/bson"
)

// helpers to build db update commands

// GetUpdateFieldValueCommand creates update command for a DocContent removing excluded fields
// if includeFields is not empty, only the fields in the list will be included
func GetUpdateDocCommand[T types.DocContent](i T, includeFields []string, excludeFields ...string) (bson.D, error) {
	m, err := flatbson.Flatten(i)
	if err != nil {
		return nil, err
	}

	for _, f := range excludeFields {
		delete(m, f)
	}
	if len(includeFields) > 0 {
		for k := range m {
			found := false
			for _, f := range includeFields {
				if strings.HasPrefix(k, f) {
					found = true
					break
				}
			}
			if !found {
				delete(m, k)
			}
		}
	}
	if len(m) == 0 {
		return nil, NoFieldsToUpdateError{}
	}
	return bson.D{bson.E{Key: "$set", Value: m}}, nil
}

func GetUpdateAddToSetCommand(arrayFieldName string, values ...interface{}) bson.D {
	if len(values) == 1 {
		return bson.D{bson.E{Key: "$addToSet", Value: bson.D{bson.E{Key: arrayFieldName, Value: values[0]}}}}
	} else {
		return bson.D{bson.E{Key: "$addToSet", Value: bson.D{bson.E{Key: arrayFieldName, Value: bson.M{"$each": values}}}}}
	}
}

func GetUpdatePullFromSetCommand(arrayFieldName string, values ...interface{}) bson.D {
	pullConditions := make(bson.A, 0, len(values))
	for _, val := range values {
		pullConditions = append(pullConditions, val)
	}
	return bson.D{bson.E{Key: "$pull", Value: bson.D{bson.E{Key: arrayFieldName, Value: bson.D{bson.E{Key: "$in", Value: pullConditions}}}}}}
}

func GetUpdateSetFieldCommand(fieldName string, value interface{}) bson.D {
	return bson.D{bson.E{Key: "$set", Value: bson.D{bson.E{Key: fieldName, Value: value}}}}
}

func GetUpdateUnsetFieldCommand(fieldName string) bson.D {
	return bson.D{bson.E{Key: "$unset", Value: bson.D{bson.E{Key: fieldName, Value: ""}}}}
}
