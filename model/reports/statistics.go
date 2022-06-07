package reports

// ReportStatistics represents statistics for an individual specification report.
type ReportStatistics struct {
	FilesizeKB         int                  `json:"filesizeKb,omitempty" yaml:"filesizeKb,omitempty"`
	FilesizeBytes      int                  `json:"filesizeBytes,omitempty" yaml:"filesizeBytes,omitempty"`
	SpecType           string               `json:"specType,omitempty" yaml:"specType,omitempty"`
	SpecFormat         string               `json:"specFormat,omitempty" yaml:"specFormat,omitempty"`
	Version            string               `json:"version,omitempty" yaml:"version,omitempty"`
	References         int                  `json:"references,omitempty" yaml:"references,omitempty"`
	ExternalDocs       int                  `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	Schemas            int                  `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Parameters         int                  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Links              int                  `json:"links,omitempty" yaml:"links,omitempty"`
	Paths              int                  `json:"paths,omitempty" yaml:"paths,omitempty"`
	Operations         int                  `json:"operations,omitempty" yaml:"operations,omitempty"`
	Tags               int                  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Examples           int                  `json:"examples,omitempty" yaml:"examples,omitempty"`
	Enums              int                  `json:"enums,omitempty" yaml:"enums,omitempty"`
	Security           int                  `json:"security,omitempty" yaml:"security,omitempty"`
	OverallScore       int                  `json:"overallScore,omitempty" yaml:"overallScore,omitempty"`
	TotalErrors        int                  `json:"totalErrors,omitempty" yaml:"totalErrors,omitempty"`
	TotalWarnings      int                  `json:"totalWarnings,omitempty" yaml:"totalWarnings,omitempty"`
	TotalInfo          int                  `json:"totalInfo,omitempty" yaml:"totalInfo,omitempty"`
	TotalHints         int                  `json:"totalHints,omitempty" yaml:"totalHints,omitempty"`
	CategoryStatistics []*CategoryStatistic `json:"categoryStatistics,omitempty" yaml:"categoryStatistics,omitempty"`
}

// CategoryStatistic represents the number of issues for a particular category
type CategoryStatistic struct {
	CategoryName string `json:"categoryName" yaml:"categoryName"`
	CategoryId   string `json:"categoryId" yaml:"categoryId"`
	NumIssues    int    `json:"numIssues" yaml:"numIssues"`
	Score        int    `json:"score" yaml:"score"`
	Warnings     int    `json:"warnings" yaml:"warnings"`
	Errors       int    `json:"errors" yaml:"errors"`
	Info         int    `json:"info" yaml:"info"`
	Hints        int    `json:"hints" yaml:"hints"`
}
