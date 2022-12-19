package db

import (
	"config-service/db/mongo"
	"config-service/types"
	"config-service/utils/consts"
	"config-service/utils/log"
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	mongoDB "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//////////////////////////////////Sugar functions for mongo using values in gin context /////////////////////////////////////////
/////////////////////////////////all methods are expecting collection and customerGUID from context/////////////////////////////

// GetAllForCustomer returns all docs for customer
func GetAllForCustomer[T any](c context.Context, includeGlobals bool) ([]T, error) {
	return GetAllForCustomerWithProjection[T](c, nil, includeGlobals)
}

// GetAllForCustomerWithProjection returns all docs for customer with projection
func GetAllForCustomerWithProjection[T any](c context.Context, projection bson.D, includeGlobals bool) ([]T, error) {
	defer log.LogNTraceEnterExit("GetAllForCustomerWithProjection", c)()
	collection, _, err := ReadContext(c)
	result := []T{}
	if err != nil {
		return nil, err
	}
	fb := NewFilterBuilder()
	if includeGlobals {
		fb.WithNotDeleteForCustomerAndGlobal(c)
	} else {
		fb.WithNotDeleteForCustomer(c)
	}
	filter := fb.Get()
	findOpts := options.Find().SetNoCursorTimeout(true)
	if projection != nil {
		findOpts.SetProjection(projection)
	}
	if cur, err := mongo.GetReadCollection(collection).
		Find(c, filter, findOpts); err != nil {
		return nil, err
	} else if err := cur.All(c, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func FindForCustomer[T any](c context.Context, filterBuilder *FilterBuilder, projection bson.D) ([]T, error) {
	defer log.LogNTraceEnterExit("FindForCustomer", c)()
	collection, _, err := ReadContext(c)
	result := []T{}
	if err != nil {
		return nil, err
	}
	if filterBuilder == nil {
		filterBuilder = NewFilterBuilder()
	}
	filter := filterBuilder.WithNotDeleteForCustomer(c).Get()
	findOpts := options.Find().SetNoCursorTimeout(true)
	if projection != nil {
		findOpts.SetProjection(projection)
	}
	if cur, err := mongo.GetReadCollection(collection).
		Find(c, filter, findOpts); err != nil {
		return nil, err
	} else {

		if err := cur.All(c, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// UpdateDocument updates document by GUID and update command
func UpdateDocument[T any](c context.Context, id string, update bson.D) ([]T, error) {
	defer log.LogNTraceEnterExit("UpdateDocument", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	var oldDoc T
	if err := mongo.GetReadCollection(collection).
		FindOne(c,
			NewFilterBuilder().
				WithNotDeleteForCustomer(c).
				WithID(id).
				Get()).
		Decode(&oldDoc); err != nil {
		if err == mongoDB.ErrNoDocuments {
			return nil, nil
		}
		log.LogNTraceError("failed to get document by id", err, c)
		return nil, err
	}
	var newDoc T
	filter := NewFilterBuilder().WithNotDeleteForCustomer(c).WithID(id).Get()
	if err := mongo.GetWriteCollection(collection).FindOneAndUpdate(c, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&newDoc); err != nil {
		return nil, err
	}
	return []T{oldDoc, newDoc}, nil
}

// DocExist returns true if at least one document with given filter exists
func DocExist(c context.Context, f bson.D) (bool, error) {
	defer log.LogNTraceEnterExit("DocExist", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return false, err
	}
	filter := NewFilterBuilder().
		WithNotDeleteForCustomer(c).
		WithFilter(f).
		Get()
	n, err := mongo.GetReadCollection(collection).CountDocuments(c, filter, options.Count().SetLimit(1))
	return n > 0, err
}

// DocWithNameExist returns true if at least one document with given name exists
func DocWithNameExist(c context.Context, name string) (bool, error) {
	defer log.LogNTraceEnterExit("DocWithNameExist", c)()
	return DocExist(c,
		NewFilterBuilder().
			WithName(name).
			Get())
}

// GetDocByGUID returns document by GUID owned by customer
func GetDocByGUID[T any](c context.Context, guid string) (*T, error) {
	defer log.LogNTraceEnterExit("GetDocByGUID", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	var result T
	if err := mongo.GetReadCollection(collection).
		FindOne(c,
			NewFilterBuilder().
				WithNotDeleteForCustomer(c).
				WithGUID(guid).
				Get()).
		Decode(&result); err != nil {
		if err == mongoDB.ErrNoDocuments {
			return nil, nil
		}
		log.LogNTraceError("failed to get document by id", err, c)
		return nil, err
	}
	return &result, nil
}

// GetDo returns document by given filter
func GetDoc[T any](c context.Context, filter *FilterBuilder) (*T, error) {
	defer log.LogNTraceEnterExit("GetDocByGUID", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	var result T
	bfilter := bson.D{}
	if filter != nil {
		bfilter = filter.Get()
	}
	if err := mongo.GetReadCollection(collection).
		FindOne(c, bfilter).
		Decode(&result); err != nil {
		if err == mongoDB.ErrNoDocuments {
			return nil, nil
		}
		log.LogNTraceError("failed to get document by id", err, c)
		return nil, err
	}
	return &result, nil
}

// GetDocByName returns document by name
func GetDocByName[T any](c context.Context, name string) (*T, error) {
	defer log.LogNTraceEnterExit("GetDocByName", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	var result T
	if err := mongo.GetReadCollection(collection).
		FindOne(c,
			NewFilterBuilder().
				WithNotDeleteForCustomer(c).
				WithName(name).
				Get()).
		Decode(&result); err != nil {
		if err == mongoDB.ErrNoDocuments {
			return nil, nil
		}
		log.LogNTraceError("failed to get document by name", err, c)
		return nil, err
	}
	return &result, nil
}

// CountDocs counts documents that match the filter
func CountDocs(c context.Context, f bson.D) (int64, error) {
	defer log.LogNTraceEnterExit("CountDocs", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return 0, err
	}
	filter := NewFilterBuilder().
		WithNotDeleteForCustomer(c).
		WithFilter(f).
		Get()
	return mongo.GetReadCollection(collection).CountDocuments(c, filter)
}

func InsertDBDocument[T types.DocContent](c context.Context, dbDoc types.Document[T]) (T, error) {
	defer log.LogNTraceEnterExit("InsertDBDocument", c)()
	collection, err := readCollection(c)
	if err != nil {
		return nil, err
	}
	if _, err := mongo.GetWriteCollection(collection).InsertOne(c, dbDoc); err != nil {
		return nil, err
	} else {
		return dbDoc.Content, nil
	}
}

func InsertDocuments[T types.DocContent](c context.Context, docs []T) ([]T, error) {
	defer log.LogNTraceEnterExit("InsertDocuments", c)()
	collection, customerGUID, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	dbDocs := []interface{}{}
	for i := range docs {
		dbDocs = append(dbDocs, types.NewDocument(docs[i], customerGUID))
	}

	if len(dbDocs) == 1 {
		if _, err := mongo.GetWriteCollection(collection).InsertOne(c, dbDocs[0]); err != nil {
			return nil, err
		} else {
			return docs, nil
		}
	} else {
		if _, err := mongo.GetWriteCollection(collection).InsertMany(c, dbDocs); err != nil {
			return nil, err
		} else {
			return docs, nil
		}
	}
}

func DeleteByName[T types.DocContent](c context.Context, name string) (deletedDoc *T, err error) {
	defer log.LogNTraceEnterExit("DeleteByName", c)()
	collection, err := readCollection(c)
	if err != nil {
		return nil, err
	}
	toBeDeleted, err := GetDocByName[T](c, name)
	if err != nil {
		return nil, err
	} else if toBeDeleted == nil {
		return nil, nil
	}

	if res, err := mongo.GetWriteCollection(collection).DeleteOne(c, bson.M{consts.IdField: (*toBeDeleted).GetGUID()}); err != nil {
		return nil, err
	} else if res.DeletedCount == 0 {
		return nil, nil
	}
	return toBeDeleted, nil
}

func DeleteByGUID[T types.DocContent](c context.Context, guid string) (deletedDoc *T, err error) {
	defer log.LogNTraceEnterExit("DeleteByGUID", c)()
	collection, err := readCollection(c)
	if err != nil {
		return nil, err
	}
	toBeDeleted, err := GetDocByGUID[T](c, guid)
	if err != nil {
		return nil, err
	} else if toBeDeleted == nil {
		return nil, nil
	}
	if res, err := mongo.GetWriteCollection(collection).DeleteOne(c, bson.M{consts.IdField: guid}); err != nil {
		return nil, err
	} else if res.DeletedCount == 0 {
		return nil, nil
	}
	return toBeDeleted, nil
}

func BulkDeleteByName[T types.DocContent](c context.Context, names []string) (deletedCount int64, err error) {
	defer log.LogNTraceEnterExit("BulkDeleteByName", c)()
	collection, err := readCollection(c)
	if err != nil {
		return 0, err
	}
	filter := NewFilterBuilder().WithIn("name", names).WithNotDeleteForCustomer(c)
	if res, err := mongo.GetWriteCollection(collection).DeleteMany(c, filter.Get()); err != nil {
		return 0, err
	} else {
		return res.DeletedCount, nil
	}
}

// helpers

// ReadContext reads collection and customerGUID from context
func ReadContext(c context.Context) (collection, customerGUID string, err error) {
	collection, errCollection := readCollection(c)
	if errCollection != nil {
		err = multierror.Append(err, errCollection)
	}
	customerGUID, errGuid := readCustomerGUID(c)
	if errGuid != nil {
		err = multierror.Append(err, errGuid)
	}
	return collection, customerGUID, err
}

func readCustomerGUID(c context.Context) (customerGUID string, err error) {
	if val := c.Value(consts.CustomerGUID); val != nil {
		customerGUID = val.(string)
	}
	if customerGUID == "" {
		err = fmt.Errorf("customerGUID is not in context")
	}
	return customerGUID, err
}

func readCollection(c context.Context) (collection string, err error) {
	if val := c.Value(consts.Collection); val != nil {
		collection = val.(string)
	}
	if collection == "" {
		err = fmt.Errorf("collection is not in context")
	}
	return collection, err
}

func IsDuplicateKeyError(err error) bool {
	return mongoDB.IsDuplicateKeyError(err)
}
