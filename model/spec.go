package model

// SpecInfo represents information about a supplied specification.
type SpecInfo struct {
	SpecType   string `json:"type"`
	Version    string `json:"version"`
	SpecFormat string `json:"format"`
}

// SearchResult represents the position of a result in a specification.
type SearchResult struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}
