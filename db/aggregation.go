package db

import (
	"config-service/db/mongo"
	"config-service/types"
	"config-service/utils"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"config-service/utils/log"
	"text/template"

	"github.com/armosec/armoapi-go/armotypes"
	mongoDB "go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

const MaxAggregationLimit = 10000

type preDefinedQuery string

const (
	CustomersWithScansBetweenDates preDefinedQuery = "customersWithScansBetweenDates"
)

var rootTemplate = template.New("root")

//go:embed predefined_queries/customersWithScansBetweenDates.txt
var CustomersWithScansBetweenDatesBytes string

func Init() {
	t := rootTemplate.New(string(CustomersWithScansBetweenDates))
	template.Must(t.Parse(CustomersWithScansBetweenDatesBytes))
}

type Metadata struct {
	Total    int `json:"total" bson:"total"`
	Limit    int `json:"limit" bson:"limit"`
	NextSkip int `json:"nextSkip" bson:"nextSkip"`
}

type AggResult[T any] struct {
	Metadata Metadata `json:"metadata" bson:"metadata"`
	Results  []T      `json:"results" bson:"results"`
}

type aggResponse[T any] struct {
	Metadata []Metadata `json:"metadata" bson:"metadata"`
	Results  []T        `json:"results" bson:"results"`
}

func AggregateWithTemplate[T any](ctx context.Context, limit, cursor int, collection string, queryTemplateName preDefinedQuery, templateArgs map[string]interface{}) (*AggResult[T], error) {
	msg := fmt.Sprintf("AggregateWithTemplate collection %s queryTemplateName %s  templateArgs %v", collection, queryTemplateName, templateArgs)
	log.LogNTraceEnterExit(msg, ctx)()
	if templateArgs == nil {
		templateArgs = map[string]interface{}{}
	}
	templateArgs["skip"] = cursor
	if limit == 0 || limit > MaxAggregationLimit {
		limit = MaxAggregationLimit
	}
	templateArgs["limit"] = limit
	buf := strings.Builder{}
	if err := rootTemplate.ExecuteTemplate(&buf, string(queryTemplateName), templateArgs); err != nil {
		log.LogNTraceError("failed to execute template", err, ctx)
		return nil, err
	}
	var pipeline []bson.M
	if err := json.Unmarshal([]byte(buf.String()), &pipeline); err != nil {
		log.LogNTraceError("failed to unmarshal template", err, ctx)
		return nil, err
	}
	dbCursor, err := mongo.GetReadCollection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		log.LogNTraceError("failed aggregate", err, ctx)
		return nil, err
	}

	resultsSlice := []aggResponse[T]{}
	if err := dbCursor.All(ctx, &resultsSlice); err != nil {
		log.LogNTraceError("failed to decode results", err, ctx)
		return nil, err
	}
	results := AggResult[T]{}
	if len(resultsSlice) == 0 {
		return &results, nil
	}
	if len(resultsSlice[0].Metadata) != 0 {
		results.Metadata = resultsSlice[0].Metadata[0]
	}
	results.Metadata.Limit = limit
	results.Results = resultsSlice[0].Results
	if cursor+len(results.Results) < results.Metadata.Total {
		results.Metadata.NextSkip = cursor + len(results.Results)
	}

	return &results, nil
}

func uniqueValuePipeline(field string, match bson.D, skip, limit int64, schemaInfo types.SchemaInfo) mongoDB.Pipeline {
	isArray, arrayPath, _ := schemaInfo.IsArrayPath(field)
	filedRef := "$" + field
	pipeline := mongoDB.Pipeline{
		{{Key: "$match", Value: match}},
	}
	if isArray {
		pipeline = append(pipeline,
			bson.D{{Key: "$unwind", Value: "$" + arrayPath}},
			//after unwind we need to match again
			bson.D{{Key: "$match", Value: match}},
		)
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: filedRef},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{
			{Key: "_id", Value: 1},
		}}},
	)
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: skip}})
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$limit", Value: limit}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "values", Value: bson.D{{Key: "$push", Value: "$_id"}}},
			{Key: "count", Value: bson.D{{Key: "$push", Value: bson.D{{Key: "key", Value: "$_id"}, {Key: "count", Value: "$count"}}}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "values", Value: 1},
			{Key: "count", Value: 1},
		}}},
	)
	return pipeline
}

func addUniqueValuesResult(aggregatedResults *armotypes.UniqueValuesResponseV2, field string, result aggregateResult) {
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
}
