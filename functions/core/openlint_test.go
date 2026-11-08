// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package core

import (
	"testing"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestOpenLintTruthValuesPreserveNative(t *testing.T) {
	for _, tc := range []struct {
		source           string
		native, openlint bool
	}{
		{"{}", true, false}, {"[]", true, false}, {"'false'", true, false}, {"'0'", true, false},
		{"false", true, true}, {"0", true, true}, {"null", true, true}, {"''", true, true}, {"text", false, false},
	} {
		t.Run(tc.source, func(t *testing.T) {
			var node yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.source), &node))
			require.Equal(t, tc.native, falsyForContract(node.Content[0], model.NativeContract))
			require.Equal(t, tc.openlint, falsyForContract(node.Content[0], model.OpenLintV1Draft))
		})
	}
}

func TestOpenLintRegexSemanticsAndIsolation(t *testing.T) {
	for _, contract := range []model.Contract{model.NativeContract, model.OpenLintV1Draft} {
		t.Run(string(contract), func(t *testing.T) {
			t.Parallel()
			match, err := compilePattern(`^\s$`, model.RuleFunctionContext{Contract: contract})
			require.NoError(t, err)
			for _, value := range []string{"\u00a0", "\v"} {
				matched, err := match.matchString(value)
				require.NoError(t, err)
				require.Equal(t, contract == model.OpenLintV1Draft, matched)
			}
		})
	}
	match, err := compilePattern(`(?<=a)b`, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
	require.NoError(t, err)
	matched, err := match.matchString("ab")
	require.NoError(t, err)
	require.True(t, matched)
	_, err = compilePattern(`(?<=a)b`, model.RuleFunctionContext{})
	require.Error(t, err)
}

func TestOpenLintRegexDeadline(t *testing.T) {
	match, err := compilePattern(`^(a+)+$`, model.RuleFunctionContext{Contract: model.OpenLintV1Draft, RuleTimeout: time.Millisecond})
	require.NoError(t, err)
	matched, err := match.matchString("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!")
	require.Error(t, err)
	require.False(t, matched)
}

func openLintTestContext(function string, options map[string]any) model.RuleFunctionContext {
	rule := buildCoreTestRule("$.value", model.SeverityWarn, function, "", options)
	return model.RuleFunctionContext{Contract: model.OpenLintV1Draft, Rule: &rule, RuleAction: model.CastToRuleAction(rule.Then), Given: "$.value", Options: options}
}

func TestOpenLintPresentAndAbsentValues(t *testing.T) {
	for _, tc := range []struct {
		name             string
		function         model.RuleFunction
		missing, present int
	}{
		{"truthy", &Truthy{}, 1, 0}, {"falsy", Falsy{}, 0, 1}, {"defined", Defined{}, 1, 0}, {"undefined", Undefined{}, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := openLintTestContext(tc.name, nil)
			require.Len(t, tc.function.RunRule([]*yaml.Node{nil}, ctx), tc.missing)
			require.Len(t, tc.function.RunRule([]*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}, ctx), tc.present)
		})
	}
}

func TestOpenLintPatternFlagsAndCodeUnits(t *testing.T) {
	for _, tc := range []struct {
		pattern, value string
		match          bool
	}{
		{`/^abc$/i`, "ABC", true}, {`/^b/m`, "a\nb", true}, {`/a.b/s`, "a\nb", true},
		{`/^.$/`, "😀", false}, {`/^..$/`, "😀", true}, {`/^.$/u`, "😀", true}, {`/^😀$/`, "😀", true},
		{`/b/y`, "ab", false}, {`/b/g`, "ab", true}, {`/^\s$/`, "\u00a0", true},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			match, err := compilePattern(tc.pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
			require.NoError(t, err)
			got, err := match.matchString(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.match, got)
		})
	}
	for _, pattern := range []string{`/x/ii`, `/x/q`} {
		_, err := compilePattern(pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
		require.Error(t, err)
	}
	ctx := openLintTestContext("pattern", map[string]any{"notMatch": ""})
	require.Len(t, (Pattern{}).RunRule([]*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "anything"}}, ctx), 1)
	require.Empty(t, (Pattern{}).RunRule([]*yaml.Node{{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "anything"}}}}, ctx))
}

func TestOpenLintLengthBounds(t *testing.T) {
	for _, tc := range []struct {
		source  string
		options map[string]any
		count   int
	}{
		{`"😀"`, map[string]any{"max": 1}, 1}, {`"😀"`, map[string]any{"max": 2}, 0},
		{`[1]`, map[string]any{"max": 0}, 1}, {`[]`, map[string]any{"max": 0}, 0},
		{`-1`, map[string]any{"min": 0}, 1}, {`1.5`, map[string]any{"max": 1.25}, 1},
		{`false`, map[string]any{"min": 2}, 0}, {`null`, map[string]any{"min": 2}, 0},
		{`{one: 1}`, map[string]any{"min": 2}, 1},
	} {
		t.Run(tc.source, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.source), &root))
			require.Len(t, (Length{}).RunRule(root.Content, openLintTestContext("length", tc.options)), tc.count)
		})
	}
}

func TestOpenLintXorProperties(t *testing.T) {
	for _, tc := range []struct {
		source string
		count  int
	}{
		{`{first: null}`, 0}, {`{first: false, third: null}`, 1}, {`{nested: {third: null}}`, 1}, {`text`, 0},
	} {
		t.Run(tc.source, func(t *testing.T) {
			var root yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(tc.source), &root))
			require.Len(t, (Xor{}).RunRule(root.Content, openLintTestContext("xor", map[string]any{"properties": []any{"first", "second", "third"}})), tc.count)
		})
	}
}

func TestOpenLintOptionValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options any
	}{
		{"truthy", map[string]any{"unexpected": true}},
		{"casing", map[string]any{"type": "lower"}},
		{"casing", map[string]any{"type": "camel", "separator": map[string]any{"char": "ab"}}},
		{"length", map[string]any{"max": "1"}},
		{"pattern", map[string]any{"match": 42}},
		{"pattern", map[string]any{"match": "["}},
		{"enumeration", map[string]any{"values": []any{map[string]any{}}}},
		{"xor", map[string]any{"properties": []any{"a", 3}}},
		{"schema", map[string]any{"schema": map[string]any{}, "allErrors": "true"}},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Error(t, ValidateOpenLintOptions(tc.name, tc.options)) })
	}
	require.NoError(t, ValidateOpenLintOptions("truthy", nil))
	require.NoError(t, ValidateOpenLintOptions("pattern", map[string]any{"match": ""}))
	require.NoError(t, ValidateOpenLintOptions("xor", map[string]any{"properties": []any{"a", "b", "c"}}))
	require.NoError(t, ValidateOpenLintOptions("schema", map[string]any{"schema": map[string]any{}, "dialect": "auto"}))
	require.NoError(t, ValidateOpenLintOptions("custom", map[string]any{"any": true}))
}

func TestOpenLintSchemaReportsMissingPropertiesSeparately(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("{}"), &root))
	opts := map[string]any{"schema": map[string]any{"type": "object", "required": []any{"a", "b"}}, "allErrors": true}
	ctx := openLintTestContext("schema", opts)
	got := (Schema{}).RunRule(root.Content, ctx)
	require.Len(t, got, 2)
	require.Contains(t, got[0].Message, "a")
	require.Contains(t, got[1].Message, "b")
	ctx.Contract = model.NativeContract
	require.Len(t, (Schema{}).RunRule(root.Content, ctx), 1)
	delete(opts, "allErrors")
	ctx.Contract = model.OpenLintV1Draft
	require.Len(t, (Schema{}).RunRule(root.Content, ctx), 1)
}

func TestOpenLintSchemaDialectPrecedence(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("[42]"), &root))
	opts := map[string]any{"schema": map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "array", "prefixItems": []any{map[string]any{"type": "string"}}}, "dialect": "draft7"}
	ctx := openLintTestContext("schema", opts)
	require.Empty(t, (Schema{}).RunRule(root.Content, ctx))
	opts["dialect"] = "auto"
	require.Len(t, (Schema{}).RunRule(root.Content, ctx), 1)
}

func TestOpenLintAlphabeticalDoesNotCompareAcrossTypes(t *testing.T) {
	for _, source := range []string{`[z, null, a]`, `[{rank: 2}, {rank: 10}]`} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))
		opts := map[string]any{}
		if source[1] == '{' {
			opts["keyedBy"] = "rank"
		}
		ctx := openLintTestContext("alphabetical", opts)
		require.Empty(t, (Alphabetical{}).RunRule(root.Content, ctx))
		ctx.Contract = model.NativeContract
		require.Len(t, (Alphabetical{}).RunRule(root.Content, ctx), 1)
	}
}

func TestOpenLintPatternIdentityEscapes(t *testing.T) {
	for _, tc := range []struct {
		pattern, value string
		matches        bool
	}{
		{`\A`, "", false}, {`\A`, "A", true},
		{`\z`, "", false}, {`\z`, "z", true},
		{`\Z`, "", false}, {`\Z`, "Z", true},
		{`\p{L}`, "a", false}, {`\p{L}`, "p{L}", true},
		{`/\p{L}/u`, "a", true},
		{`/a/my`, "\na", false}, {`/a/my`, "a", true},
		{`/😀/uy`, "a😀", false}, {`/😀/uy`, "😀", true},
	} {
		t.Run(tc.pattern+"_"+tc.value, func(t *testing.T) {
			matcher, err := compilePattern(tc.pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
			require.NoError(t, err)
			matched, err := matcher.matchString(tc.value)
			require.NoError(t, err)
			require.Equal(t, tc.matches, matched)
		})
	}
	// The native contract keeps Go's beginning-of-input anchor.
	matcher, err := compilePattern(`\A`, model.RuleFunctionContext{})
	require.NoError(t, err)
	matched, err := matcher.matchString("")
	require.NoError(t, err)
	require.True(t, matched)
}

func TestOpenLintRejectsNonJavaScriptGroups(t *testing.T) {
	for _, pattern := range []string{
		`(?i)a`, `(?=a)(?i)a`, `(a)\1(?i)b`, `(?<=a)(?i)b`,
		`(?>a)`, `(?#comment)a`, `(?(1)a|b)`, `(?P<name>a)`, `(?'name'a)`,
		`(?<name>a)(?<-name>b)`, `(?)`, `(?n:a)`, `(?x:a)`,
		`(?ii:a)`, `(?i-i:a)`, `(?i--m:a)`, `(?-:a)`,
		`[(]a(?i)b`, `(?<name>(?i)a)`,
	} {
		t.Run(pattern, func(t *testing.T) {
			_, err := compilePattern(pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
			require.Error(t, err)
		})
	}
	for _, pattern := range []string{
		`(?=a)a`, `(?<=a)b`, `(a)\1`, `(?<name>a)\k<name>`,
		`\(\?i\)`, `[(?i)]`, `(?i:a)`, `(?im-s:a)`, `(?-i:a)`, `(?i-:a)`,
		`(?<名字>a)`, `(?<name>a)(?=b)b`,
	} {
		t.Run(pattern, func(t *testing.T) {
			_, err := compilePattern(pattern, model.RuleFunctionContext{Contract: model.OpenLintV1Draft})
			require.NoError(t, err)
		})
	}
	// Native patterns retain Go's global inline flag syntax.
	matcher, err := compilePattern(`(?i)a`, model.RuleFunctionContext{})
	require.NoError(t, err)
	matched, err := matcher.matchString("A")
	require.NoError(t, err)
	require.True(t, matched)
}

func TestNativePatternIgnoresEmptyNotMatch(t *testing.T) {
	ctx := openLintTestContext("pattern", map[string]any{"notMatch": ""})
	ctx.Contract = model.NativeContract
	node := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "anything"}
	require.Empty(t, (Pattern{}).RunRule([]*yaml.Node{node}, ctx))
	ctx.Contract = model.OpenLintV1Draft
	require.Len(t, (Pattern{}).RunRule([]*yaml.Node{node}, ctx), 1)
}
