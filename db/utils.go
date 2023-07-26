package db

import (
	"config-service/db/mongo"
	"config-service/types"
	"config-service/utils"
	"config-service/utils/consts"
	"config-service/utils/log"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/armosec/armoapi-go/armotypes"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	mongoDB "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

// triggers collection actions - called by the router builder on startup
func ValidateCollection(collection string) error {
	return mongo.IndexCollection(collection)
}

//////////////////////////////////Sugar functions for mongo using values in gin context /////////////////////////////////////////
/////////////////////////////////all methods are expecting collection and customerGUID from context/////////////////////////////

// GetAllForCustomer returns all docs for customer
func GetAllForCustomer[T any](c context.Context, includeGlobals bool) ([]T, error) {
	findOps := NewFindOptions()
	if includeGlobals {
		findOps.Filter().WithNotDeleteForCustomerAndGlobal(c)
	} else {
		findOps.Filter().WithNotDeleteForCustomer(c)
	}
	return AdminFind[T](c, findOps)
}

func FindForCustomerWithGlobals[T any](c context.Context, findOpts *FindOptions) ([]T, error) {
	defer log.LogNTraceEnterExit("FindForCustomerWithGlobals", c)()
	if findOpts == nil {
		findOpts = NewFindOptions()
	}
	findOpts.Filter().WithNotDeleteForCustomerAndGlobal(c)
	return AdminFind[T](c, findOpts)
}

func FindForCustomer[T any](c context.Context, findOpts *FindOptions) ([]T, error) {
	defer log.LogNTraceEnterExit("FindForCustomer", c)()
	if findOpts == nil {
		findOpts = NewFindOptions()
	}
	findOpts.Filter().WithNotDeleteForCustomer(c)
	return AdminFind[T](c, findOpts)
}

// AdminFind search for docs of all customers (unless filtered by caller)
func AdminFind[T any](c context.Context, findOps *FindOptions) ([]T, error) {
	defer log.LogNTraceEnterExit(fmt.Sprintf("AdminFind %+v", findOps), c)()
	collection, _, err := ReadContext(c)
	result := []T{}
	if err != nil {
		return nil, err
	}
	if findOps == nil {
		findOps = NewFindOptions()
	}
	dbFindOptions := options.Find().SetNoCursorTimeout(true)
	dbFindOptions.SetProjection(findOps.projection.get())
	dbFindOptions.SetSort(findOps.sort.get())
	dbFindOptions.SetSkip(int64(findOps.skip))

	if cur, err := mongo.GetReadCollection(collection).
		Find(c, findOps.filter.get(), dbFindOptions); err != nil {
		return nil, err
	} else {

		if err := cur.All(c, &result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func FindPaginatedForCustomer[T any](c context.Context, findOps *FindOptions) (*types.SearchResult[T], error) {
	defer log.LogNTraceEnterExit(fmt.Sprintf("FindPaginatedForCustomer %+v", findOps), c)()
	if findOps == nil {
		findOps = &FindOptions{}
	}
	findOps.Filter().WithNotDeleteForCustomer(c)
	return AdminFindPaginated[T](c, findOps)
}

// AdminFindPaginated search for docs of all customers (unless filtered by caller) and return paginated result
func AdminFindPaginated[T any](c context.Context, findOps *FindOptions) (*types.SearchResult[T], error) {
	defer log.LogNTraceEnterExit(fmt.Sprintf("AdminFindPaginated %+v", findOps), c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	if findOps == nil {
		findOps = &FindOptions{}
	}

	resultsPipe := []bson.M{}
	if findOps.Sort().Len() > 0 {
		resultsPipe = append(resultsPipe, bson.M{"$sort": findOps.sort.get()})
	}
	if findOps.skip > 0 {
		resultsPipe = append(resultsPipe, bson.M{"$skip": findOps.skip})
	}
	if findOps.limit > 0 {
		resultsPipe = append(resultsPipe, bson.M{"$limit": findOps.limit})
	}
	if findOps.Projection().Len() > 0 {
		resultsPipe = append(resultsPipe, bson.M{"$project": findOps.projection.get()})
	}

	pipeline := mongoDB.Pipeline{
		{{Key: "$match", Value: findOps.filter.get()}},
		{{Key: "$facet", Value: bson.M{
			"limitedResults": resultsPipe,
			"count": []bson.M{
				{"$count": "count"},
			},
		}}},
	}
	cursor, err := mongo.GetReadCollection(collection).Aggregate(c, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(c)
	var result paginatedResult[T]
	if cursor.Next(c) {
		err = cursor.Decode(&result)
		if err != nil {
			return nil, err
		}
	}
	searchRes := &types.SearchResult[T]{}
	var count int64
	if len(result.Count) > 0 {
		count = result.Count[0].Count
	}
	searchRes.SetCount(count)
	searchRes.SetResults(result.LimitedResults)
	return searchRes, nil
}

func AggregateForCustomer[T any](c context.Context, findOps *FindOptions) (*armotypes.UniqueValuesResponseV2, error) {
	defer log.LogNTraceEnterExit(fmt.Sprintf("AggregateForCustomer %+v", findOps), c)()
	if findOps == nil {
		findOps = &FindOptions{}
	}
	findOps.Filter().WithNotDeleteForCustomer(c)
	return AdminAggregate[T](c, findOps)
}

// AdminAggregate search for docs of all customers (unless filtered by caller) and return aggregated result
func AdminAggregate[T any](c context.Context, findOps *FindOptions) (*armotypes.UniqueValuesResponseV2, error) {
	defer log.LogNTraceEnterExit(fmt.Sprintf("AdminAggregate %+v", findOps), c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	if findOps == nil {
		return nil, fmt.Errorf("findOps is nil")
	}
	if len(findOps.group) == 0 {
		return nil, fmt.Errorf("group is empty")
	}
	//store each field result in a sync map
	results := sync.Map{}
	errGroup, ctx := errgroup.WithContext(c)
	for _, field := range findOps.group {
		field := field
		fieldFilter := NewFilterBuilder().WithFilter(findOps.filter).AddExists(field, true)
		filedRef := "$" + field
		errGroup.Go(func() error {
			pipeline := mongoDB.Pipeline{
				{{Key: "$match", Value: fieldFilter.get()}},
				{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: filedRef},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
				{{Key: "$sort", Value: bson.D{
					{Key: "_id", Value: 1},
				}}},
				{{Key: "$limit", Value: findOps.limit}},
				{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "values", Value: bson.D{{Key: "$push", Value: "$_id"}}},
					{Key: "count", Value: bson.D{{Key: "$push", Value: bson.D{{Key: "key", Value: "$_id"}, {Key: "count", Value: "$count"}}}}},
				}}},
				{{Key: "$project", Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "values", Value: 1},
					{Key: "count", Value: 1},
				}}},
			}
			cursor, err := mongo.GetReadCollection(collection).Aggregate(ctx, pipeline)
			if err != nil {
				return fmt.Errorf("failed to aggregate field %s: %w", field, err)
			}
			defer cursor.Close(ctx)
			var result aggregateResult
			if cursor.Next(ctx) {
				err = cursor.Decode(&result)
				if err != nil {
					return fmt.Errorf("failed to decode field %s: %w", field, err)
				}
			}
			results.Store(field, result)
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}

	//aggregate all fields results into one response
	aggregatedResults := &armotypes.UniqueValuesResponseV2{
		Fields:      make(map[string][]string),
		FieldsCount: make(map[string][]armotypes.UniqueValuesResponseFieldsCount),
	}
	var aggregateResultErr error
	results.Range(func(key, value interface{}) bool {
		field, ok := key.(string)
		if !ok {
			aggregateResultErr = errors.New("failed to cast key")
			return false
		}
		result, ok := value.(aggregateResult)
		if !ok {
			aggregateResultErr = fmt.Errorf("failed to cast result for field %s", field)
			return false
		}
		aggregatedResults.Fields[field] = make([]string, len(result.Values))
		for i, value := range result.Values {
			aggregatedResults.Fields[field][i] = utils.Interface2String(value)
		}
		aggregatedResults.FieldsCount[field] = []armotypes.UniqueValuesResponseFieldsCount{}
		for _, count := range result.Count {
			aggregatedResults.FieldsCount[field] = append(aggregatedResults.FieldsCount[field], armotypes.UniqueValuesResponseFieldsCount{
				Field: count.Key,
				Count: count.Count,
			})
		}
		return true
	})
	return aggregatedResults, aggregateResultErr
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
				get()).
		Decode(&oldDoc); err != nil {
		if err == mongoDB.ErrNoDocuments {
			return nil, nil
		}
		log.LogNTraceError("failed to get document by id", err, c)
		return nil, err
	}
	var newDoc T
	filter := NewFilterBuilder().WithNotDeleteForCustomer(c).WithID(id).get()
	if err := mongo.GetWriteCollection(collection).FindOneAndUpdate(c, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).
		Decode(&newDoc); err != nil {
		return nil, err
	}
	return []T{oldDoc, newDoc}, nil
}

func AddToArray(c context.Context, id string, arrayPath string, values ...interface{}) (modified int64, err error) {
	defer log.LogNTraceEnterExit("AddToArray", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return 0, err
	}
	//filter documents that already have this value in the array
	filter := NewFilterBuilder().WithNotDeleteForCustomer(c).WithID(id).get()

	update := GetUpdateAddToSetCommand(arrayPath, values...)
	res, err := mongo.GetWriteCollection(collection).UpdateOne(c, filter, update)
	if res != nil {
		modified = res.ModifiedCount
	}
	return modified, err
}

func UpdateOne(c context.Context, id string, update interface{}) (modified int64, err error) {
	defer log.LogNTraceEnterExit("UpdateOne", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return 0, err
	}
	filterBuilder := NewFilterBuilder().WithNotDeleteForCustomer(c).WithID(id)
	res, err := mongo.GetWriteCollection(collection).UpdateOne(c, filterBuilder.get(), update)
	if res != nil {
		modified = res.ModifiedCount
	}
	return modified, err
}

func PullFromArray(c context.Context, id string, arrayPath string, values ...interface{}) (modified int64, err error) {
	defer log.LogNTraceEnterExit("PullFromArray", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return 0, err
	}
	filterBuilder := NewFilterBuilder().WithNotDeleteForCustomer(c).WithID(id)
	update := GetUpdatePullFromSetCommand(arrayPath, values...)
	res, err := mongo.GetWriteCollection(collection).UpdateOne(c, filterBuilder.get(), update)
	if res != nil {
		modified = res.ModifiedCount
	}
	return modified, err
}

// DocExist returns true if at least one document with given filter exists
func DocExist(c context.Context, filterBuilder *FilterBuilder) (bool, error) {
	defer log.LogNTraceEnterExit("DocExist", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return false, err
	}
	filter := NewFilterBuilder().
		WithNotDeleteForCustomer(c).
		WithFilter(filterBuilder).
		get()
	n, err := mongo.GetReadCollection(collection).CountDocuments(c, filter, options.Count().SetLimit(1))
	return n > 0, err
}

// DocWithNameExist returns true if at least one document with given name exists
func DocWithNameExist(c context.Context, name string) (bool, error) {
	defer log.LogNTraceEnterExit("DocWithNameExist", c)()
	return DocExist(c,
		NewFilterBuilder().
			WithName(name))
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
				WithID(guid).
				get()).
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
	defer log.LogNTraceEnterExit("GetDoc", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return nil, err
	}
	var result T
	bfilter := bson.D{}
	if filter != nil {
		bfilter = filter.get()
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
				get()).
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
func CountDocs(c context.Context, filterBuilder *FilterBuilder) (int64, error) {
	defer log.LogNTraceEnterExit("CountDocs", c)()
	collection, _, err := ReadContext(c)
	if err != nil {
		return 0, err
	}
	filter := NewFilterBuilder().
		WithNotDeleteForCustomer(c).
		WithFilter(filterBuilder).
		get()
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
	filter := NewFilterBuilder().WithIn("name", names)
	return BulkDelete[T](c, *filter)
}

func BulkDelete[T types.DocContent](c context.Context, filter FilterBuilder) (deletedCount int64, err error) {
	defer log.LogNTraceEnterExit("BulkDelete", c)()
	collection, err := readCollection(c)
	if err != nil {
		return 0, err
	}
	filter.WithNotDeleteForCustomer(c)
	if res, err := mongo.GetWriteCollection(collection).DeleteMany(c, filter.get()); err != nil {
		return 0, err
	} else {
		return res.DeletedCount, nil
	}
}
func DeleteCustomerDocs(c context.Context) (deletedCount int64, err error) {
	defer log.LogNTraceEnterExit("DeleteAllCustomerDocs", c)()
	customerGUID, err := readCustomerGUID(c)
	if err != nil {
		return 0, err
	}
	return AdminDeleteCustomersDocs(c, customerGUID)
}

func AdminDeleteCustomersDocs(c context.Context, customerGUIDs ...string) (deletedCount int64, err error) {
	defer log.LogNTraceEnterExit("AdminDeleteAllCustomerDocs", c)()
	if len(customerGUIDs) == 0 {
		return 0, nil
	}
	collections, err := mongo.ListCollectionNames(c)
	if err != nil {
		return 0, err
	}

	var deletionErrs error
	errChanel := make(chan error, len(collections))
	errWg := sync.WaitGroup{}
	errWg.Add(1)
	go func() {
		defer errWg.Done()
		for err := range errChanel {
			deletionErrs = multierror.Append(deletionErrs, err)
		}
	}()

	//delete the customers themselves
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(customerGUIDs []string) {
		defer wg.Done()
		idsFilter := NewFilterBuilder().WithIDs(customerGUIDs)
		res, err := mongo.GetWriteCollection(consts.CustomersCollection).DeleteMany(c, idsFilter.get())
		if err != nil {
			errChanel <- err
		}
		if res != nil {
			atomic.AddInt64(&deletedCount, res.DeletedCount)
		}
	}(customerGUIDs)

	//delete all the customers docs in all collections
	ownersFilter := NewFilterBuilder().WithCustomers(customerGUIDs)
	for _, collection := range collections {
		if collection == consts.CustomersCollection {
			continue
		}
		wg.Add(1)
		go func(collection string, customerGUIDs []string) {
			defer wg.Done()
			res, err := mongo.GetWriteCollection(collection).DeleteMany(c, ownersFilter.get())
			if err != nil {
				log.LogNTraceError(fmt.Sprintf("AdminDeleteAllCustomerDocs errors when deleting documents in collection:%s", collection), err, c)
				errChanel <- err
			}
			if res != nil {
				atomic.AddInt64(&deletedCount, res.DeletedCount)
				log.LogNTrace(fmt.Sprintf("AdminDeleteAllCustomerDocs deleted %d documents in collection:%s", res.DeletedCount, collection), c)
			}
		}(collection, customerGUIDs)

	}
	wg.Wait()
	close(errChanel)
	errWg.Wait()
	return atomic.LoadInt64(&deletedCount), deletionErrs
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

func IsNoFieldsToUpdateError(err error) bool {
	return errors.Is(err, NoFieldsToUpdateError{})
}

type NoFieldsToUpdateError struct {
}

func (e NoFieldsToUpdateError) Error() string {
	return "no fields to update"
}
