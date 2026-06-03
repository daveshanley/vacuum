package statistics

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// CalculateQualityScore calculates the quality score based on the number of errors, warnings, and info messages.
// This is the single source of truth for score calculation logic.
// Returns a score between 10 and 100, where:
// - Info messages deduct 0.1 points each
// - Warnings deduct 0.3 points each
// - Errors deduct 15 points each
// - oas3-schema violations deduct an additional 90 points
// - Minimum score is 10 (never returns 0 or negative)
// - If no errors but score would be negative, floor at 25
func CalculateQualityScore(resultSet *model.RuleResultSet) int {
	if resultSet == nil {
		return 100 // perfect score if no results
	}

	total := 100.0
	score := total - float64(resultSet.GetInfoCount())*0.1
	score = score - (0.3 * float64(resultSet.GetWarnCount()))
	score = score - (15.0 * float64(resultSet.GetErrorCount())) // errors are failures they should be judged harshly.

	if resultSet.GetErrorCount() <= 0 && score < 0 {
		// floor at 25% if there are no errors, but a ton of warnings lowering the score
		score = 25.0
	}

	// if there are any document-schema rule violations, bottom out the score,
	// an invalid API description is a big deal.
	for _, result := range resultSet.Results {
		if result.Rule != nil && (result.Rule.Id == "oas3-schema" ||
			result.Rule.Id == "asyncapi-3-document-resolved" ||
			result.Rule.Id == "asyncapi-3-document-unresolved") {
			score = score - 90
		}
	}

	if score <= 0 {
		score = 10 // the lowest score we want to present can't be 0, there has to be some hope!
	}

	return int(score)
}

// CreateReportStatistics generates a ready to render breakdown of the document's statistics. A convenience function
// that reduces churn on building stats over and over for different interfaces.
func CreateReportStatistics(index *index.SpecIndex, info *datamodel.SpecInfo, results *model.RuleResultSet) *reports.ReportStatistics {

	// don't go looking for stats if we don't have the necessary data
	if index == nil || info == nil || results == nil {
		return nil
	}

	opPCount := index.GetOperationsParameterCount()
	cPCount := index.GetComponentParameterCount()

	var catStats []*reports.CategoryStatistic
	for _, cat := range model.RuleCategoriesOrdered {
		var numIssues, numWarnings, numErrors, numInfo, numHints int
		numIssues = len(results.GetResultsByRuleCategory(cat.Id))
		numWarnings = len(results.GetWarningsByRuleCategory(cat.Id))
		numErrors = len(results.GetErrorsByRuleCategory(cat.Id))
		numInfo = len(results.GetInfoByRuleCategory(cat.Id))
		numHints = len(results.GetHintByRuleCategory(cat.Id))
		numResults := len(results.Results)
		var score int
		if numResults == 0 && numIssues == 0 {
			score = 100 // perfect
		} else if numResults > 0 {
			score = 100 - (numIssues * 100 / numResults)
		}
		catStats = append(catStats, &reports.CategoryStatistic{
			CategoryName: cat.Name,
			CategoryId:   cat.Id,
			NumIssues:    numIssues,
			Warnings:     numWarnings,
			Errors:       numErrors,
			Info:         numInfo,
			Hints:        numHints,
			Score:        score,
		})
	}

	// Use the shared score calculation function
	overallScore := CalculateQualityScore(results)

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
		OverallScore:       overallScore,
		TotalErrors:        results.GetErrorCount(),
		TotalWarnings:      results.GetWarnCount(),
		TotalInfo:          results.GetInfoCount(),
		CategoryStatistics: catStats,
	}
	if isAsyncAPIInfo(info) {
		applyAsyncAPIStatistics(stats, index, info)
	}
	return stats
}

func isAsyncAPIInfo(info *datamodel.SpecInfo) bool {
	if info == nil {
		return false
	}
	return strings.EqualFold(info.SpecType, "asyncapi") || strings.HasPrefix(info.SpecFormat, model.AsyncAPI3)
}

func applyAsyncAPIStatistics(stats *reports.ReportStatistics, specIndex *index.SpecIndex, info *datamodel.SpecInfo) {
	if stats == nil || info == nil {
		return
	}
	root := statsRoot(info.RootNode)
	if root == nil {
		return
	}

	components := statsMappingValue(root, "components")
	stats.Paths = 0
	stats.Channels = statsMappingLen(statsMappingValue(root, "channels")) + statsMappingLen(statsMappingValue(components, "channels"))
	stats.Operations = statsMappingLen(statsMappingValue(root, "operations")) + statsMappingLen(statsMappingValue(components, "operations"))
	stats.Messages = statsMappingLen(statsMappingValue(components, "messages")) +
		asyncAPIChannelMessages(statsMappingValue(root, "channels")) +
		asyncAPIChannelMessages(statsMappingValue(components, "channels"))
	stats.Servers = statsMappingLen(statsMappingValue(root, "servers")) + statsMappingLen(statsMappingValue(components, "servers"))
	stats.Schemas = statsMappingLen(statsMappingValue(components, "schemas"))
	stats.Security = statsMappingLen(statsMappingValue(components, "securitySchemes"))

	accumulator := asyncAPIStatsAccumulator{countReferences: specIndex == nil}
	accumulator.walk(root)
	stats.Parameters = statsMappingLen(statsMappingValue(components, "parameters")) +
		asyncAPIChannelParameters(statsMappingValue(root, "channels")) +
		asyncAPIChannelParameters(statsMappingValue(components, "channels"))
	stats.Replies = statsMappingLen(statsMappingValue(components, "replies")) + accumulator.replies
	stats.Tags = accumulator.tags
	stats.Bindings = accumulator.bindings
	if specIndex != nil {
		stats.References = len(specIndex.GetMappedReferences())
	} else {
		stats.References = accumulator.references
	}
}

func statsRoot(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func statsMappingValue(node *yaml.Node, key string) *yaml.Node {
	node = statsRoot(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func statsMappingLen(node *yaml.Node) int {
	if node == nil || node.Kind != yaml.MappingNode {
		return 0
	}
	return len(node.Content) / 2
}

func asyncAPIChannelMessages(channels *yaml.Node) int {
	if channels == nil || channels.Kind != yaml.MappingNode {
		return 0
	}
	total := 0
	for i := 1; i < len(channels.Content); i += 2 {
		messages := statsMappingValue(channels.Content[i], "messages")
		if messages != nil && messages.Kind == yaml.SequenceNode {
			total += len(messages.Content)
		}
	}
	return total
}

func asyncAPIChannelParameters(channels *yaml.Node) int {
	if channels == nil || channels.Kind != yaml.MappingNode {
		return 0
	}
	total := 0
	for i := 1; i < len(channels.Content); i += 2 {
		total += statsMappingLen(statsMappingValue(channels.Content[i], "parameters"))
	}
	return total
}

type asyncAPIStatsAccumulator struct {
	replies         int
	tags            int
	bindings        int
	references      int
	countReferences bool
}

func (a *asyncAPIStatsAccumulator) walk(node *yaml.Node) {
	node = statsRoot(node)
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content)-1; i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1]
			switch key {
			case "$ref":
				if a.countReferences && value.Value != "" {
					a.references++
				}
			case "reply":
				a.replies++
			case "tags":
				if value.Kind == yaml.SequenceNode {
					a.tags += len(value.Content)
				}
			case "bindings":
				a.bindings += statsMappingLen(value)
			}
			a.walk(value)
		}
		return
	}
	for _, child := range node.Content {
		a.walk(child)
	}
}
