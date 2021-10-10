package model

type SpecInfo struct {
	SpecType   string `json:"type"`
	Version    string `json:"version"`
	SpecFormat string `json:"format"`
}

type SearchResult struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}
