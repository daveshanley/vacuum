package reports

// ReportStatistics represents statistics for an individual specification report.
type ReportStatistics struct {
	FilesizeKB         int                  `json:"filesizeKb"`
	FilesizeBytes      int                  `json:"filesizeBytes"`
	SpecType           string               `json:"specType"`
	SpecFormat         string               `json:"specFormat"`
	Version            string               `json:"version"`
	References         int                  `json:"references"`
	ExternalDocs       int                  `json:"externalDocs"`
	Schemas            int                  `json:"schemas"`
	Parameters         int                  `json:"parameters"`
	Links              int                  `json:"links"`
	Paths              int                  `json:"paths"`
	Operations         int                  `json:"operations"`
	Tags               int                  `json:"tags"`
	Examples           int                  `json:"examples"`
	Enums              int                  `json:"enums"`
	Security           int                  `json:"security"`
	OverallScore       int                  `json:"overallScore"`
	TotalErrors        int                  `json:"totalErrors"`
	TotalWarnings      int                  `json:"totalWarnings"`
	TotalInfo          int                  `json:"totalInfo"`
	TotalHints         int                  `json:"totalHints"`
	CategoryStatistics []*CategoryStatistic `json:"categoryStatistics"`
}

// CategoryStatistic represents the number of issues for a particular category
type CategoryStatistic struct {
	CategoryName string `json:"categoryName"`
	CategoryId   string `json:"categoryId"`
	NumIssues    int    `json:"numIssues"`
	Score        int    `json:"score"`
	Warnings     int    `json:"warnings"`
	Errors       int    `json:"errors"`
	Info         int    `json:"info"`
	Hints        int    `json:"hints"`
}
