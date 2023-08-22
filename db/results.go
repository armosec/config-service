package db

type paginatedResult[T any] struct {
	Count          []countResult `bson:"count"`
	LimitedResults []T           `bson:"limitedResults"`
}
type countResult struct {
	Count int64 `bson:"count"`
}

type aggregateResult struct {
	Values []interface{} `bson:"values"`
	Count  []struct {
		Key   interface{} `bson:"key"`
		Count int64       `bson:"count"`
	} `bson:"count"`
}
