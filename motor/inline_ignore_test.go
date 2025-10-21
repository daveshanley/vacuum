package motor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestCheckInlineIgnore_SingleRule(t *testing.T) {
	spec := `
info:
  title: Test
  x-lint-ignore: ["rule-id"]
  description: missing
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	infoNode := node.Content[0].Content[1]

	ignored := checkInlineIgnore(infoNode, "rule-id")
	assert.True(t, ignored)

	ignored = checkInlineIgnore(infoNode, "other-rule")
	assert.False(t, ignored)
}

func TestCheckInlineIgnore_ArrayOfRules(t *testing.T) {
	spec := `
info:
  title: Test
  x-lint-ignore: 
    - rule-one
    - rule-two
  description: missing
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	infoNode := node.Content[0].Content[1]

	assert.True(t, checkInlineIgnore(infoNode, "rule-one"))
	assert.True(t, checkInlineIgnore(infoNode, "rule-two"))
	assert.False(t, checkInlineIgnore(infoNode, "rule-three"))
}

func TestCheckInlineIgnore_StringRule(t *testing.T) {
	spec := `
info:
  title: Test
  x-lint-ignore: single-rule
  description: missing
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	infoNode := node.Content[0].Content[1]

	assert.True(t, checkInlineIgnore(infoNode, "single-rule"))
	assert.False(t, checkInlineIgnore(infoNode, "other-rule"))
}

func TestCheckInlineIgnore_NoIgnore(t *testing.T) {
	spec := `
info:
  title: Test
  description: missing
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	infoNode := node.Content[0].Content[1]

	assert.False(t, checkInlineIgnore(infoNode, "any-rule"))
}

func TestCheckInlineIgnore_NonMappingNode(t *testing.T) {
	spec := `
info:
  description: missing
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	descNode := node.Content[0].Content[1].Content[1]

	assert.False(t, checkInlineIgnore(descNode, "any-rule"))
}

func TestInlineIgnore_Integration_InfoDescription(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  x-lint-ignore: ["info-description"]
paths: {}
`

	// Create a simple ruleset that requires info description
	rulesetYaml := `
extends: []
rules:
  info-description:
    description: Info must have a description
    given: $.info
    severity: error
    then:
      function: truthy
      field: description
`

	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(rulesetYaml))
	require.NoError(t, err)

	execution := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(spec),
		SpecFileName: "test.yaml",
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have no regular results (rule was ignored)
	assert.Len(t, result.Results, 0)

	// Should have one ignored result
	assert.Len(t, result.IgnoredResults, 1)
	assert.Equal(t, "info-description", result.IgnoredResults[0].RuleId)
	assert.Equal(
		t,
		"Rule ignored due to inline ignore directive",
		result.IgnoredResults[0].Message,
	)
}

func TestInlineIgnore_Integration_NoIgnore(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`

	// Create a simple ruleset that requires info description
	rulesetYaml := `
extends: []
rules:
  info-description:
    description: Info must have a description
    given: $.info
    severity: error
    then:
      function: truthy
      field: description
`

	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(rulesetYaml))
	require.NoError(t, err)

	execution := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(spec),
		SpecFileName: "test.yaml",
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have one regular result (rule was not ignored)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "info-description", result.Results[0].RuleId)

	// Should have no ignored results
	assert.Len(t, result.IgnoredResults, 0)
}

func TestInlineIgnore_Integration_ArrayOfRules(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  x-lint-ignore: 
    - info-description
    - info-contact
paths: {}
`

	// Create a ruleset with multiple rules
	rulesetYaml := `
extends: []
rules:
  info-description:
    description: Info must have a description
    given: $.info
    severity: error
    then:
      function: truthy
      field: description
  info-contact:
    description: Info must have contact
    given: $.info
    severity: error
    then:
      function: truthy
      field: contact
  info-license:
    description: Info must have license
    given: $.info
    severity: error
    then:
      function: truthy
      field: license
`

	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(rulesetYaml))
	require.NoError(t, err)

	execution := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(spec),
		SpecFileName: "test.yaml",
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have one regular result (info-license was not ignored)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "info-license", result.Results[0].RuleId)

	// Should have two ignored results
	assert.Len(t, result.IgnoredResults, 2)

	ignoredRuleIds := make([]string, len(result.IgnoredResults))
	for i, ignored := range result.IgnoredResults {
		ignoredRuleIds[i] = ignored.RuleId
	}

	assert.Contains(t, ignoredRuleIds, "info-description")
	assert.Contains(t, ignoredRuleIds, "info-contact")
}

func TestInlineIgnore_Integration_PathLevel(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    x-lint-ignore: ["path-rule"]
    get:
      summary: Get users
  /posts:
    get:
      summary: Get posts
`

	// Create a ruleset that targets paths
	rulesetYaml := `
extends: []
rules:
  path-rule:
    description: Path must have description
    given: $.paths[*]
    severity: error
    then:
      function: truthy
      field: description
`

	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(rulesetYaml))
	require.NoError(t, err)

	execution := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(spec),
		SpecFileName: "test.yaml",
	}

	result := ApplyRulesToRuleSet(execution)

	// Should have one regular result (/posts path was not ignored)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, "path-rule", result.Results[0].RuleId)

	// Should have one ignored result (/users path was ignored)
	assert.Len(t, result.IgnoredResults, 1)
	assert.Equal(t, "path-rule", result.IgnoredResults[0].RuleId)
}

func TestFilterIgnoreNodes(t *testing.T) {
	spec := `
info:
  title: Test API
  x-lint-ignore: ["rule-id"]
  description: Test description
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(spec), &node)
	require.NoError(t, err)

	infoNode := node.Content[0].Content[1]
	allNodes := infoNode.Content

	var testNodes []*yaml.Node
	testNodes = append(testNodes, allNodes...)

	filtered := filterIgnoreNodes(testNodes)

	// Should have filtered out the ignore key and its value
	assert.Len(t, filtered, 4)

	for _, filteredNode := range filtered {
		if filteredNode.Kind == yaml.ScalarNode {
			assert.NotEqual(t, ignoreKey, filteredNode.Value)
		}
	}
}

func TestIsIgnoreNode(t *testing.T) {
	ignoreKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: ignoreKey}
	assert.True(t, isIgnoreNode(ignoreKeyNode))

	regularKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "title"}
	assert.False(t, isIgnoreNode(regularKeyNode))

	assert.False(t, isIgnoreNode(nil))

	mappingNode := &yaml.Node{Kind: yaml.MappingNode}
	assert.False(t, isIgnoreNode(mappingNode))
}

func TestInlineIgnore_Integration_CustomRuleFiltering(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
  x-lint-ignore: ["info-description"]
  description: Test description
paths: {}
`

	// Create a custom rule that targets all properties under info (including ignore key)
	rulesetYaml := `
extends: []
rules:
  custom-info-rule:
    description: Custom rule that targets all info properties
    given: $.info.*
    severity: error
    then:
      function: truthy
`

	rc := CreateRuleComposer()
	rs, err := rc.ComposeRuleSet([]byte(rulesetYaml))
	require.NoError(t, err)

	execution := &RuleSetExecution{
		RuleSet:      rs,
		Spec:         []byte(spec),
		SpecFileName: "test.yaml",
	}

	result := ApplyRulesToRuleSet(execution)

	// Verify none of the results are for ignore key
	for _, res := range result.Results {
		if res.StartNode != nil && res.StartNode.Kind == yaml.ScalarNode {
			assert.NotEqual(
				t,
				ignoreKey,
				res.StartNode.Value,
				"ignore key should be filtered out",
			)
		}
	}

	assert.Greater(t, len(result.Results), 0, "Should have some results from the custom rule")
}
