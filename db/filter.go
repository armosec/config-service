package db

import (
	"config-service/utils/consts"
	"context"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// FilterBuilder builds filters for queries
type FilterBuilder struct {
	filter       bson.D
	customerIsID bool
}

func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filter: bson.D{},
	}
}

func (f *FilterBuilder) WithFilter(filterBuilder *FilterBuilder) *FilterBuilder {
	filter := filterBuilder.get()
	f.filter = append(f.filter, filter...)
	return f
}

func (f *FilterBuilder) Len() int {
	return len(f.filter)
}

func (f *FilterBuilder) get() bson.D {
	return f.filter
}

func (f *FilterBuilder) WithGlobal() *FilterBuilder {
	return f.WithValue(consts.CustomersField, "")
}

func (f *FilterBuilder) WithID(id string) *FilterBuilder {
	idFound := false
	if f.customerIsID {
		for i := range f.filter {
			if f.filter[i].Key == consts.IdField {
				f.filter[i].Value = id
				idFound = true
				return f
			}
		}
	}
	if idFound {
		return f
	}
	return f.WithValue(consts.IdField, id)
}

func (f *FilterBuilder) WithIDs(ids []string) *FilterBuilder {
	if f.customerIsID {
		for i := range f.filter {
			if f.filter[i].Key == consts.IdField {
				f.filter[i].Value = ids
				return f
			}
		}
	}
	return f.WithIn(consts.IdField, ids)
}

func (f *FilterBuilder) WithName(name string) *FilterBuilder {
	return f.WithValue(consts.NameField, name)
}

func (f *FilterBuilder) WithCustomer(c context.Context) *FilterBuilder {
	customerGUID, _ := c.Value(consts.CustomerGUID).(string)
	if collection, _ := c.Value(consts.Collection).(string); collection == consts.CustomersCollection {
		f.customerIsID = true
		return f.WithID(customerGUID)
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

func (f *FilterBuilder) WithValue(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: value})
	return f
}

func (f *FilterBuilder) WithRegex(key string, value string, ignoreCase bool) *FilterBuilder {
	regexValue := bson.D{{Key: "$regex", Value: value}}
	if ignoreCase {
		regexValue = append(regexValue, bson.E{Key: "$options", Value: "i"})
	}
	f.filter = append(f.filter, bson.E{Key: key, Value: regexValue})
	return f
}

func (f *FilterBuilder) WithRange(key string, minVal, maxVal interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$gte", Value: minVal}, {Key: "$lte", Value: maxVal}}})
	return f
}

func (f *FilterBuilder) WithGreaterThanEqual(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$gte", Value: value}}})
	return f
}

func (f *FilterBuilder) WithLowerThanEqual(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$lte", Value: value}}})
	return f
}

func (f *FilterBuilder) WithNotEqual(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$ne", Value: value}}})
	return f
}

func (f *FilterBuilder) WithEqual(key string, value interface{}) *FilterBuilder {
	f.filter = append(f.filter, bson.E{Key: key, Value: bson.D{{Key: "$eq", Value: value}}})
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

func (f *FilterBuilder) AddExists(key string, value bool) *FilterBuilder {
	if value {
		// Field exists
		f.filter = append(f.filter, bson.E{
			Key:   key,
			Value: bson.M{"$exists": true, "$ne": nil},
		})
	} else {
		// Field does not exist or is null
		f.filter = append(f.filter, bson.E{
			Key: "$or",
			Value: []interface{}{
				bson.M{key: bson.M{"$exists": false}},
				bson.M{key: nil},
			},
		})
	}
	return f
}

func (f *FilterBuilder) AddOr(filters ...*FilterBuilder) *FilterBuilder {
	orM := []bson.M{}
	for _, filter := range filters {
		m := bson.M{}
		for i := range filter.filter {
			m[filter.filter[i].Key] = filter.filter[i].Value
		}
		orM = append(orM, m)
	}
	f.filter = append(f.filter, bson.E{Key: "$or", Value: orM})
	return f
}

func (f *FilterBuilder) AddAnd(filters ...*FilterBuilder) *FilterBuilder {
	orM := []bson.M{}
	for _, filter := range filters {
		m := bson.M{}
		for i := range filter.filter {
			m[filter.filter[i].Key] = filter.filter[i].Value
		}
		orM = append(orM, m)
	}
	f.filter = bson.D{{Key: "$and", Value: orM}}
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
