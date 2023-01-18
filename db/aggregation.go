package db

import (
	"config-service/db/mongo"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"config-service/utils/log"
	"text/template"

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
