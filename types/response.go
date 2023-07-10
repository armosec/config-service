package types

type SearchResult[T any] struct {
	Total   int64 `json:"total"`
	Results []T   `json:"results"`
}
