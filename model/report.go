package model

type Report struct {
	Code     string   `json:"code,omitempty"`
	Path     []string `json:"path,omitempty"`
	Message  string   `json:"message,omitempty"`
	Severity int      `json:"severity,omitempty"`
	Range    Range    `json:"range,omitempty"`
}

type Range struct {
	Start Context `json:"start,omitempty"`
	End   Context `json:"end,omitempty"`
}

type Context struct {
	Line      int `json:"line,omitempty"`
	Character int `json:"character,omitempty"`
}
