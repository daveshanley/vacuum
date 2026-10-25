package motor

import (
	"fmt"
	"testing"

	"github.com/pb33f/testify/require"
)

func TestJavaScriptCompatibilityPackages(t *testing.T) {
	for _, prefix := range []string{"@stoplight/spectral-", "@openlint/openlint-"} {
		t.Run(prefix, func(t *testing.T) {
			rs := loadJSRules(t, fmt.Sprintf(`
import {createRulesetFunction} from '%score';
import {truthy} from '%sfunctions';
import {oas3} from '%sformats';
const check = createRulesetFunction({input:null, options:null}, async (value, options, context) => truthy(value, options, context));
export default {formats:[oas3], rules:{check:{given:'$.info', severity:'error', then:{field:'title', function:check}}}};
`, prefix, prefix, prefix))
			for _, tc := range []struct {
				name, spec string
				findings   int
			}{
				{"valid", "openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}", 0},
				{"missing", "openapi: 3.0.3\ninfo: {version: '1'}\npaths: {}", 1},
				{"excluded", "swagger: '2.0'\ninfo: {version: '1'}\npaths: {}", 0},
			} {
				t.Run(tc.name, func(t *testing.T) {
					result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte(tc.spec), SilenceLogs: true})
					defer result.Release()
					require.Empty(t, result.Errors)
					require.Len(t, result.Results, tc.findings)
					if tc.findings > 0 {
						require.Equal(t, "check", result.Results[0].Rule.Id)
						require.Equal(t, "error", result.Results[0].Rule.Severity)
						require.Equal(t, "$.info.title", result.Results[0].Path)
						require.Equal(t, "value must be defined", result.Results[0].Message)
					}
				})
			}
		})
	}
}

func TestJavaScriptCompatibilityOr(t *testing.T) {
	for _, prefix := range []string{"@stoplight/spectral-", "@openlint/openlint-"} {
		for _, callable := range []string{"or", "(value, options, context) => or(value, options, context)"} {
			t.Run(prefix+callable, func(t *testing.T) {
				rs := loadJSRules(t, fmt.Sprintf(`import {or} from '%sfunctions';
export default {rules:{check:{given:'$.info', then:{function:%s, functionOptions:{properties:['title','description']}}}}};`, prefix, callable))
				for _, tc := range []struct {
					info     string
					findings int
				}{
					{"{version: '1'}", 1},
					{"{version: '1', title: null}", 0},
					{"{version: '1', title: false}", 0},
					{"{version: '1', description: ''}", 0},
				} {
					t.Run(tc.info, func(t *testing.T) {
						result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: " + tc.info + "\npaths: {}"), SilenceLogs: true})
						defer result.Release()
						require.Empty(t, result.Errors)
						require.Len(t, result.Results, tc.findings)
						if tc.findings > 0 {
							require.Equal(t, "$.info", result.Results[0].Path)
							require.Equal(t, `At least one of "title" or "description" must be defined`, result.Results[0].Message)
						}
					})
				}
			})
		}
	}
}

func TestJavaScriptCompatibilityOrRejectsInvalidOptions(t *testing.T) {
	for _, prefix := range []string{"@stoplight/spectral-", "@openlint/openlint-"} {
		t.Run(prefix, func(t *testing.T) {
			direct := loadJSRules(t, fmt.Sprintf(`import {or} from '%sfunctions';export default {rules:{check:{given:'$.info',then:{function:or,functionOptions:{properties:['title']}}}}};`, prefix))
			require.NotEmpty(t, ValidateRuleSet(direct, nil))
			wrapped := loadJSRules(t, fmt.Sprintf(`import {or} from '%sfunctions';export default {rules:{check:{given:'$.info',then:{function:(v,o,c)=>or(v,o,c),functionOptions:{properties:['title']}}}}};`, prefix))
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: wrapped, Spec: []byte("openapi: 3.0.3\ninfo: {version: '1'}\npaths: {}"), SilenceLogs: true})
			defer result.Release()
			require.NotEmpty(t, result.Errors)
			require.Empty(t, result.Results)
		})
	}
}

func TestJavaScriptCompatibilityUnreferenced(t *testing.T) {
	for _, prefix := range []string{"@stoplight/spectral-", "@openlint/openlint-"} {
		for _, callable := range []string{"unreferencedReusableObject", "(v,o,c)=>unreferencedReusableObject(v,o,c)"} {
			t.Run(prefix+callable, func(t *testing.T) {
				rs := loadJSRules(t, fmt.Sprintf(`import {unreferencedReusableObject} from '%sfunctions';export default {rules:{unused:{resolved:false,given:'$.defs',then:{function:%s,functionOptions:{reusableObjectsLocation:'#/defs'}}}}};`, prefix, callable))
				result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("use: {$ref: '#/defs/Used'}\ndefs: {Used: {}, Unused: {}}"), SkipDocumentCheck: true, SilenceLogs: true})
				defer result.Release()
				require.Empty(t, result.Errors)
				require.Len(t, result.Results, 1)
				require.Equal(t, "$.defs.Unused", result.Results[0].Path)
			})
		}
	}
}
