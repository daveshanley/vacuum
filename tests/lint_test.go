package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/suite"
)

type LintTest struct {
	suite.Suite
}

func TestLint(t *testing.T) {
	suite.Run(t, new(LintTest))
}

func printResult(violation *model.RuleFunctionResult) {
	fmt.Printf("Rule: %s\n", violation.Rule.Id)
	// iterate over each violation of this rule
	// print out the start line, column, violation message.
	fmt.Printf(" - %s\n %s\n", violation.Message, violation.Path)
}

func printResultSet(resultSet *model.RuleResultSet) {

	for _, violation := range resultSet.Results {
		printResult(violation)
	}
}

func (s *LintTest) validateResultsMatch(rs *model.RuleResultSet, ruleId string, expectedErrorPaths []string) { // Check that all errors are accounted for
	gotErrorPaths := []string{}
	for _, result := range rs.Results {
		if result.Rule.Id != ruleId {
			continue
		}
		gotErrorPaths = append(gotErrorPaths, result.Path)
	}
	s.ElementsMatchf(expectedErrorPaths, gotErrorPaths, "did not find expected error for rule %s", ruleId)
}

func (s *LintTest) lint(specPath string) *model.RuleResultSet {
	base := filepath.Dir(specPath)
	specBytes, err := os.ReadFile(specPath)
	s.NoError(err)

	rulesetBytes, err := os.ReadFile("./test_data/path_resolving/rules.yaml")
	s.NoError(err)

	ruleset, err := rulesets.CreateRuleSetFromData(rulesetBytes)
	s.NoError(err)

	defaultRuleSets := rulesets.BuildDefaultRuleSets()
	ruleset = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(ruleset)

	lintingResults := motor.ApplyRulesToRuleSet(
		&motor.RuleSetExecution{
			RuleSet:                       ruleset,
			Spec:                          specBytes,
			Base:                          base,
			Timeout:                       time.Hour,
			AllowLookup:                   true,
			BuildDeepGraph:                true,
			ExtractReferencesSequentially: true,
			BuildGraph:                    true,
		})

	resultSet := model.NewRuleResultSet(lintingResults.Results)

	resultSet.SortResultsByLineNumber()

	return resultSet
}

func (s *LintTest) validateResults(rs *model.RuleResultSet, ruleId string, expectedErrorPaths []string) { // Check that all errors are accounted for
	gotErrorPaths := []string{}
	for _, result := range rs.Results {
		if result.Rule.Id != ruleId {
			continue
		}
		gotErrorPaths = append(gotErrorPaths, result.Path)
	}

	s.ElementsMatchf(expectedErrorPaths, gotErrorPaths, "did not find expected error for rule %s", ruleId)
}

func (s *LintTest) TestNonDeterministicPathResolving() {
	resultSet := s.lint("./test_data/path_resolving/failing_lint.yaml")

	//
	s.validateResultsMatch(resultSet, "arrays-define-max-items-undefined", []string{
		// Non-deterministic path resolving.. we get one of the following results:
		// "$.components.schemas['ListPostsResponse'].properties['not_data']",
		// "$.paths['/beta/posts'].get.responses['200'].content['application/json'].schema.properties['not_data']",
	})

	s.validateResultsMatch(resultSet, "http-204-has-no-content", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].delete.responses['204']",
	})

	s.validateResultsMatch(resultSet, "paths-kebab-case", []string{
		"$.paths['/v1/exampleNotKebab']",
	})

	s.validateResultsMatch(resultSet, "paths-starts-with-major-version", []string{
		"$.paths['/beta/posts']",
		"$.paths['/alpha/posts/{channel}/{postId}']",
	})

	s.validateResultsMatch(resultSet, "paths-without-maturity-info", []string{
		"$.paths['/beta/posts']",
		"$.paths['/alpha/posts/{channel}/{postId}']",
	})

	s.validateResultsMatch(resultSet, "response-bodies-are-json", []string{
		"$.paths['/v1/example/{example_id}'].get.responses['200'].content['application/xml']",
	})

	s.validateResultsMatch(resultSet, "response-bodies-are-typed", []string{
		"$.paths['/v1/example/{example_id}'].get.responses['200'].content['application/xml'].schema.type",
	})

	s.validateResultsMatch(resultSet, "path-query-parameters-camel-cased", []string{
		"$.paths['/v1/example/{example_id}'].get.parameters[0]",
		"$.paths['/v1/example/{example_id}'].patch.parameters[0]",
	})

	s.validateResultsMatch(resultSet, "response-parameters-camel-cased", []string{
		// "$.components.schemas['ListPostsResponse']",
		"$.paths['/beta/posts'].get.responses['200'].content['application/json'].schema",
	})

	s.validateResultsMatch(resultSet, "response-headers-kebab-cased", []string{
		"$.paths['/v1/example/{example_id}'].get.responses['200'].headers['first_response_header']",
		"$.paths['/v1/example/{example_id}'].get.responses['200'].headers['second_response_header']",
		"$.paths['/v1/exampleNotKebab'].get.responses['200'].headers['third_response_header']",
	})

	s.validateResultsMatch(resultSet, "response-headers-hs-prefix", []string{
		"$.paths['/v1/example/{example_id}'].get.responses['200'].headers['first_response_header']",
		"$.paths['/v1/example/{example_id}'].get.responses['200'].headers['second_response_header']",
		"$.paths['/v1/exampleNotKebab'].get.responses['200'].headers['third_response_header']",
	})

	s.validateResultsMatch(resultSet, "enums-upper-case", []string{
		"$.paths['/v1/example/{example_id}'].get.parameters[1].schema",
		"$.paths['/v1/example/{example_id}'].get.parameters[1].schema",
	})

	s.validateResultsMatch(resultSet, "no-http-patch", []string{
		"$.paths['/v1/example/{example_id}']",
	})

	s.validateResultsMatch(resultSet, "delete-returns-http-204", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].delete.responses",
	})

	s.validateResultsMatch(resultSet, "http-delete-no-request-body", []string{
		// This is not resolving the path, it returns the rule's `given` value. We need to fix the rule in vacuum
		// "$.paths[*].delete",
	})

	s.validateResultsMatch(resultSet, "http-get-no-request-body", []string{
		// This is not resolving the path, it returns the rule's `given` value. We need to fix the rule in vacuum
		// "$.paths[*].get",
	})

	s.validateResultsMatch(resultSet, "http-204-has-no-content", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].delete.responses['204']",
	})

	s.validateResultsMatch(resultSet, "no-structural-polymorphism-oneOf", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].put.parameters[2].schema.oneOf[0]",
	})
	s.validateResultsMatch(resultSet, "no-structural-polymorphism-anyOf", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].put.parameters[3].schema.anyOf[0]",
	})
	s.validateResultsMatch(resultSet, "no-structural-polymorphism-allOf", []string{
		"$.paths['/alpha/posts/{channel}/{postId}'].put.parameters[4].schema.allOf[0]",
	})
}
