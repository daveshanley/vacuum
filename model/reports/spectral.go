package reports

// SpectralReport represents a model that can be deserialized into a spectral compatible output.
type SpectralReport struct {
	Code     string   `json:"code" yaml:"code"`         // the rule that was run
	Path     []string `json:"path" yaml:"path"`         // the path to the item, broken down into a slice
	Message  string   `json:"message" yaml:"message"`   // the result message
	Severity int      `json:"severity" yaml:"severity"` // the severity reported
	Range    Range    `json:"range" yaml:"range"`       // the location of the issue in the spec.
	Source   string   `json:"source" yaml:"source"`     // the source of the report.
}

// Range indicates the start and end of a report item
type Range struct {
	Start RangeItem `json:"start" yaml:"start"`
	End   RangeItem `json:"end" yaml:"end"`
}

// RangeItem indicates the line and character of a range.
type RangeItem struct {
	Line int `json:"line" yaml:"line"`
	Char int `json:"character" yaml:"character"`
}
