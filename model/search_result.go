package model

// SearchResult represents the position of a result in a specification.
type SearchResult struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}
