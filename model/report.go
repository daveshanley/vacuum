package model

// Report is a Spectral compatible Report structure.
type Report struct {
	Code     string   `json:"code,omitempty"`
	Path     []string `json:"path,omitempty"`
	Message  string   `json:"message,omitempty"`
	Severity int      `json:"severity,omitempty"`
	Range    Range    `json:"range,omitempty"`
}

// Range defines the range of where the issue starts and ends
type Range struct {
	Start Context `json:"start,omitempty"`
	End   Context `json:"end,omitempty"`
}

// Context describes the line and column for a Range in a Report
type Context struct {
	Line      int `json:"line,omitempty"`
	Character int `json:"character,omitempty"`
}
