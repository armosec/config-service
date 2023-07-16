package types

import "github.com/armosec/armoapi-go/armotypes"

type SearchResult[T any] struct {
	Total    armotypes.RespTotal `json:"total"`
	Response []T                 `json:"response"`
}

func (s *SearchResult[T]) SetCount(count int64) {
	s.Total.Relation = "eq"
	s.Total.Value = int(count)
}

func (s *SearchResult[T]) SetResults(result []T) {
	s.Response = result

}
