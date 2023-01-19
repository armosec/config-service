package db

import (
	"config-service/utils/consts"
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// FilterBuilder builds filters for queries
type FilterBuilder struct {
	filter bson.D
}

func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filter: bson.D{},
	}
}

func (f *FilterBuilder) WithFilter(filter bson.D) *FilterBuilder {
	for e := range filter {
		f.filter = append(f.filter, filter[e])
	}
	return f
}

func (f *FilterBuilder) Get() bson.D {
	return f.filter
}

func (f *FilterBuilder) WithNotDeleteForCustomer(c context.Context) *FilterBuilder {
	return f.WithCustomer(c).WithNotDeleted()
}

func (f *FilterBuilder) WithGlobalNotDelete() *FilterBuilder {
	return f.WithValue(consts.CustomersField, "").WithNotDeleted()
}

func (f *FilterBuilder) WithNotDeleteForCustomerAndGlobal(c context.Context) *FilterBuilder {
	return f.WithCustomerAndGlobal(c).WithNotDeleted()
}

func (f *FilterBuilder) WithGUID(guid string) *FilterBuilder {
	return f.WithValue(consts.GUIDField, guid)
}

func (f *FilterBuilder) WithID(id string) *FilterBuilder {
	return f.WithValue(consts.IdField, id)
}

func (f *FilterBuilder) WithIDs(ids []string) *FilterBuilder {
	return f.WithIn(consts.IdField, ids)
}

func (f *FilterBuilder) WithName(name string) *FilterBuilder {
	return f.WithValue(consts.NameField, name)
}

func (f *FilterBuilder) WithCustomer(c context.Context) *FilterBuilder {
	customerGUID, _ := c.Value(consts.CustomerGUID).(string)
	if collection, _ := c.Value(consts.Collection).(string); collection == consts.CustomersCollection {
		return f.WithGUID(customerGUID)
	}
	return f.WithValue(consts.CustomersField, customerGUID)
}

func (f *FilterBuilder) WithCustomerAndGlobal(c context.Context) *FilterBuilder {
	customerGUID, _ := c.Value(consts.CustomerGUID).(string)
	return f.WithIn(consts.CustomersField, []string{customerGUID, ""})
}

func (f *FilterBuilder) WithCustomers(customers []string) *FilterBuilder {
	return f.WithIn(consts.CustomersField, customers)
}

func (f *FilterBuilder) WithNotDeleted() *FilterBuilder {
	return f.WithNotEqual(consts.DeletedField, true)
}

func (f *FilterBuilder) WithDeleted() *FilterBuilder {
	return f.WithValue(consts.DeletedField, true)
}

func (f *FilterBuilder) WithValue(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: value})
	return f
}

func (f *FilterBuilder) WithNotEqual(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$ne", Value: value}}})
	return f
}

func (f *FilterBuilder) WithIn(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$in", Value: value}}})
	return f
}

func (f *FilterBuilder) WithNotIn(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$nin", Value: value}}})
	return f
}

func (f *FilterBuilder) WithExists(key string, value bool) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$exists", Value: value}}})
	return f
}

func (f *FilterBuilder) AddNotExists(key string) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$exists", Value: false}}})
	return f
}

func (f *FilterBuilder) WithElementMatch(element interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: "$elemMatch", Value: element})
	return f
}

func (f *FilterBuilder) WarpElementMatch() *FilterBuilder {
	f.filter = bson.D{{Key: "$elemMatch", Value: f.filter}}
	return f
}

func (f *FilterBuilder) WarpOr() *FilterBuilder {
	a := bson.A{}
	for i := range f.filter {
		a = append(a, bson.D{{Key: f.filter[i].Key, Value: f.filter[i].Value}})
	}
	f.filter = bson.D{{Key: "$or", Value: a}}
	return f
}

func (f *FilterBuilder) WarpNot() *FilterBuilder {
	f.filter = bson.D{{Key: "$not", Value: f.filter}}
	return f
}

func (f *FilterBuilder) WarpWithField(field string) *FilterBuilder {
	f.filter = bson.D{{Key: field, Value: f.filter}}
	return f
}

func (f *FilterBuilder) WrapDupKeysWithOr() *FilterBuilder {
	dupFound := false
	keys := make(map[string]bson.D)
	for i := range f.filter {
		if strings.HasPrefix(f.filter[i].Key, "$") {
			continue
		}
		keys[f.filter[i].Key] = append(keys[f.filter[i].Key], f.filter[i])
		if len(keys[f.filter[i].Key]) > 1 {
			dupFound = true
		}
	}
	if !dupFound {
		return f
	}
	newF := bson.D{}
	for k := range keys {
		if len(keys[k]) > 1 {
			a := bson.A{}
			for i := range keys[k] {
				a = append(a, bson.D{{Key: keys[k][i].Key, Value: keys[k][i].Value}})
			}
			newF = append(newF, bson.E{Key: "$or", Value: a})
		} else {
			newF = append(newF, keys[k][0])
		}
	}
	f.filter = newF
	return f
}
