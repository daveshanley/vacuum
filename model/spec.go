package model

import (
	"gopkg.in/yaml.v3"
	"time"
)

// SpecInfo represents information about a supplied specification.
type SpecInfo struct {
	SpecType           string                  `json:"type"`
	Version            string                  `json:"version"`
	SpecFormat         string                  `json:"format"`
	SpecFileType       string                  `json:"fileType"`
	RootNode           *yaml.Node              `json:"-"`     // reference to the root node of the spec.
	SpecBytes          *[]byte                 `json:"bytes"` // the original bytes
	SpecJSONBytes      *[]byte                 `json:"-"`     // original bytes converted to JSON
	SpecJSON           *map[string]interface{} `json:"-"`     // standard JSON map of original bytes
	Error              error                   `json:"-"`     // something go wrong?
	APISchema          string                  `json:"-"`     // API Schema for supplied spec type (2 or 3)
	Generated          time.Time               `json:"-"`
	jsonParsingChannel chan bool
}

// SearchResult represents the position of a result in a specification.
type SearchResult struct {
	Key  string `json:"key"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

// GetJSONParsingChannel returns a channel that will close once async JSON parsing is completed.
// This is required as rules may start executing before we're even done reading in the spec to JSON.
func (si SpecInfo) GetJSONParsingChannel() chan bool {
	return si.jsonParsingChannel
}
