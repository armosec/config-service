package db

import (
	"go.mongodb.org/mongo-driver/bson"
)

// SortBuilder builds projection of queries results
type SortBuilder struct {
	filter bson.D
}

func NewSortBuilder() *SortBuilder {
	return &SortBuilder{
		filter: bson.D{},
	}
}
func (f *SortBuilder) get() bson.D {
	return f.filter
}

func (f *SortBuilder) Len() int {
	return len(f.filter)
}

func (f *SortBuilder) AddAscending(key ...string) *SortBuilder {
	for _, k := range key {
		f.filter = append(f.filter, bson.E{Key: k, Value: 1})
	}
	return f
}

func (f *SortBuilder) AddDescending(key ...string) *SortBuilder {
	for _, k := range key {
		f.filter = append(f.filter, bson.E{Key: k, Value: -1})
	}
	return f
}
