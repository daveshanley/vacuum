package model

import "gopkg.in/yaml.v3"

// SpecInfo represents information about a supplied specification.
type SpecInfo struct {
	SpecType     string     `json:"type"`
	Version      string     `json:"version"`
	SpecFormat   string     `json:"format"`
	SpecFileType string     `json:"fileType"`
	RootNode     *yaml.Node `json:"-"`
	Error        error      `json:"-"` // something go wrong?
}

// SearchResult represents the position of a result in a specification.
type SearchResult struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}
