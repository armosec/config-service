package db

import (
	"config-service/utils/consts"

	"go.mongodb.org/mongo-driver/bson"
)

// ProjectionBuilder builds projection of queries results
type ProjectionBuilder struct {
	filter bson.D
}

func NewProjectionBuilder() *ProjectionBuilder {
	return &ProjectionBuilder{
		filter: bson.D{},
	}
}

func (f *ProjectionBuilder) get() bson.D {
	return f.filter
}

func (f *ProjectionBuilder) Len() int {
	return len(f.filter)
}

func (f *ProjectionBuilder) ExcludeID(key ...string) *ProjectionBuilder {
	return f.Exclude(consts.IdField)
}

func (f *ProjectionBuilder) Include(key ...string) *ProjectionBuilder {
	for _, k := range key {
		f.filter = append(f.filter, bson.E{Key: k, Value: 1})
	}
	return f
}

func (f *ProjectionBuilder) Exclude(key ...string) *ProjectionBuilder {
	for _, k := range key {
		f.filter = append(f.filter, bson.E{Key: k, Value: 0})
	}
	return f
}
