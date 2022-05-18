package statistics

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
)

// CreateReportStatistics generates a ready to render breakdown of the document's statistics. A convenience function
// that reduces churn on building stats over and over for different interfaces.
func CreateReportStatistics(index *model.SpecIndex, info *model.SpecInfo, results *model.RuleResultSet) *reports.ReportStatistics {

	opPCount := index.GetOperationsParameterCount()
	cPCount := index.GetComponentParameterCount()

	var catStats []*reports.CategoryStatistic
	for _, cat := range model.RuleCategoriesOrdered {

		catStats = append(catStats, &reports.CategoryStatistic{
			CategoryName: cat.Name,
			CategoryId:   cat.Id,
			NumIssues:    len(results.GetResultsByRuleCategory(cat.Id)),
			Warnings:     len(results.GetWarningsByRuleCategory(cat.Id)),
			Errors:       len(results.GetErrorsByRuleCategory(cat.Id)),
			Info:         len(results.GetInfoByRuleCategory(cat.Id)),
			Hints:        len(results.GetHintByRuleCategory(cat.Id)),
		})
	}

	stats := &reports.ReportStatistics{
		FilesizeBytes:      len(*info.SpecBytes),
		FilesizeKB:         len(*info.SpecBytes) / 1024,
		SpecType:           info.SpecType,
		SpecFormat:         info.SpecFormat,
		Version:            info.Version,
		References:         len(index.GetMappedReferences()),
		ExternalDocs:       len(index.GetAllExternalDocuments()),
		Schemas:            len(index.GetAllSchemas()),
		Parameters:         opPCount + cPCount,
		Links:              len(index.GetAllLinks()),
		Paths:              index.GetPathCount(),
		Operations:         index.GetOperationCount(),
		Tags:               index.GetTotalTagsCount(),
		Examples:           len(index.GetAllExamples()),
		Enums:              len(index.GetAllEnums()),
		Security:           len(index.GetAllSecuritySchemes()),
		CategoryStatistics: catStats,
	}

	return stats
}
