package motor

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestIssue964Or(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: Or test
  version: '1.0'
paths: {}
components:
  schemas:
    First:
      title: First
      type: string
    Missing:
      type: string
    Null:
      description: null
      type: string
    Both:
      title: Both
      description: Present
      type: string
    Last:
      summary: Present
      type: string
`)
	for _, options := range []string{`[title, summary, description]`, `"title, summary, description"`} {
		for _, message := range []string{"", "Custom failure", "{{error}} at {{path}}"} {
			t.Run(options+message, func(t *testing.T) {
				rules := fmt.Sprintf(`rules:
  descriptive-text:
    description: Schemas need descriptive text.
    message: %q
    given: $.components.schemas.*
    severity: error
    then:
      function: or
      functionOptions:
        properties: %s
`, message, options)
				rs, err := CreateRuleComposer().ComposeRuleSet([]byte(rules))
				require.NoError(t, err)
				result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: spec})
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, 1)
				finding := result.Results[0]
				assert.Equal(t, "$.components.schemas['Missing']", finding.Path)
				assert.Equal(t, 12, finding.StartNode.Line)
				want := `At least one of "title" or "summary" or "description" must be defined`
				if message == "Custom failure" {
					want = message
				}
				if message == "{{error}} at {{path}}" {
					want += " at /components/schemas/Missing"
				}
				assert.Equal(t, want, finding.Message)
			})
		}
	}
}

func TestIssue964OrInvalidOptions(t *testing.T) {
	for _, options := range []string{`[]`, `[title]`, `[title, 42]`, `null`, `42`} {
		t.Run(options, func(t *testing.T) {
			rs, err := CreateRuleComposer().ComposeRuleSet([]byte(fmt.Sprintf(`rules:
  invalid-or:
    given: $.info.*
    severity: error
    then:
      function: or
      functionOptions:
        properties: %s
`, options)))
			require.NoError(t, err)
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo:\n  title: Present\n  version: '1.0'\npaths: {}")})
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			assert.Contains(t, result.Results[0].Message, "'or' requires at least two property names")
		})
	}
}

func TestIssue964OrField(t *testing.T) {
	for _, tc := range []struct {
		field string
		want  int
		path  string
	}{
		{"info", 0, ""},
		{"components.schemas.Missing", 1, "$.components.schemas.Missing"},
		{"components.schemas.Absent", 0, ""},
		{"info.title", 0, ""},
	} {
		t.Run(tc.field, func(t *testing.T) {
			rs, err := CreateRuleComposer().ComposeRuleSet([]byte(fmt.Sprintf(`rules:
  descriptive-text:
    given: $
    severity: error
    then:
      field: %s
      function: or
      functionOptions:
        properties: [title, description]
`, tc.field)))
			require.NoError(t, err)
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo:\n  title: Present\n  version: '1.0'\npaths: {}\ncomponents:\n  schemas:\n    Missing:\n      type: string")})
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, tc.want)
			if tc.want != 0 {
				assert.Equal(t, tc.path, result.Results[0].Path)
				assert.Equal(t, 9, result.Results[0].StartNode.Line)
			}
		})
	}
}

func TestIssue964OrSharedSchemaPaths(t *testing.T) {
	rs, err := CreateRuleComposer().ComposeRuleSet([]byte(`rules:
  descriptive-text:
    given: $.components.schemas.*
    resolved: true
    severity: error
    then:
      function: or
      functionOptions:
        properties: [title, description]
`))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(`openapi: 3.0.3
info:
  title: References
  version: '1.0'
paths: {}
components:
  schemas:
    Shared:
      type: string
    First:
      $ref: '#/components/schemas/Shared'
    Second:
      $ref: '#/components/schemas/Shared'
`)})
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 3)
	for _, finding := range result.Results {
		assert.ElementsMatch(t, []string{
			"$.components.schemas['Shared']",
			"$.components.schemas['First']",
			"$.components.schemas['Second']",
		}, finding.Paths)
		assert.Contains(t, finding.Paths, finding.Path)
	}
}

func TestIssue964OrExtensionPaths(t *testing.T) {
	for _, tc := range []struct{ given, field, path string }{{"$", "x-settings", "$['x-settings']"}, {"$['x-settings']", "", "$['x-settings']"}} {
		rs, err := CreateRuleComposer().ComposeRuleSet([]byte(fmt.Sprintf(`rules:
  descriptive-text:
    given: %s
    then:
      field: %q
      function: or
      functionOptions:
        properties: [title, description]
`, tc.given, tc.field)))
		require.NoError(t, err)
		result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo:\n  title: Present\n  version: '1.0'\npaths: {}\nx-settings:\n  version: '1.0'")})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 1)
		assert.Equal(t, tc.path, result.Results[0].Path)
		assert.Equal(t, 7, result.Results[0].StartNode.Line)
	}
}

func TestIssue964OrYAMLAliases(t *testing.T) {
	rs, err := CreateRuleComposer().ComposeRuleSet([]byte(`rules:
  descriptive-text:
    given: $.components.schemas.*
    resolved: false
    then:
      function: or
      functionOptions:
        properties: [title, description]
`))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(`openapi: 3.0.3
info:
  title: Aliases
  version: '1.0'
paths: {}
components:
  schemas:
    Missing: &missing {type: string}
    AlsoMissing: *missing
    Present: &present {type: string, title: null}
    AlsoPresent: *present
    MergedMissing:
      <<: *missing
    MergedPresent:
      <<: *present
`)})
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 3)
	paths := make([]string, 0, len(result.Results))
	for _, finding := range result.Results {
		paths = append(paths, finding.Path)
	}
	assert.ElementsMatch(t, []string{"$.components.schemas.Missing", "$.components.schemas.AlsoMissing", "$.components.schemas.MergedMissing"}, paths)
}

func TestIssue964OrSharedResponsePaths(t *testing.T) {
	dir, specPath, spec := writeIssue879AliasedResponseFixture(t)
	rs, err := CreateRuleComposer().ComposeRuleSet([]byte(`rules:
  descriptive-text:
    given: $.paths[*][*].responses['400'].content['*/*'].schema.properties.error
    resolved: true
    then:
      function: or
      functionOptions:
        properties: [title, description]
`))
	require.NoError(t, err)
	result := ApplyRulesToRuleSet(&RuleSetExecution{
		RuleSet: rs, Spec: spec, SpecFileName: specPath, Base: dir, AllowLookup: true,
	})
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	assert.ElementsMatch(t, []string{
		"$.paths['/v1/bar'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/bar'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/baz'].post.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].get.responses['400'].content['*/*'].schema.properties['error']",
		"$.paths['/v1/foo'].post.responses['400'].content['*/*'].schema.properties['error']",
	}, result.Results[0].Paths)
}
