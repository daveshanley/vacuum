package motor

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/config"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/testify/require"
)

func loadJSRules(t *testing.T, code string) *rulesets.RuleSet {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.mjs")
	require.NoError(t, os.WriteFile(path, []byte(code), 0600))
	raw, err := rulesets.LoadLocalRuleSet(context.Background(), path)
	require.NoError(t, err)
	rs := rulesets.BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(raw)
	require.NoError(t, rs.LoadError())
	return rs
}
func TestJavaScriptRulesetExecution(t *testing.T) {
	rs := loadJSRules(t, `let calls=0;export default {rules:{check:{given:'$.info',resolved:false,then:{field:'title',function:(v,o,c)=>{calls++;return [{message:c.document.source+':'+v+':'+calls,path:c.path}]}}}}}`)
	for i := 0; i < 2; i++ {
		result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, SpecFileName: "api.yaml", Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 1)
		require.Equal(t, "api.yaml:Hello:1", result.Results[0].Message)
		require.Equal(t, "$.info.title", result.Results[0].Path)
		result.Release()
	}
}
func TestJavaScriptRulesetFailure(t *testing.T) {
	for _, body := range []string{`throw Error('broken')`, `return null`, `return [{}]`, `return Promise.reject(Error('broken'))`, `return c.documentInventory`} {
		t.Run(body, func(t *testing.T) {
			rs := loadJSRules(t, `export default {rules:{check:{given:'$',then:{function:(v,o,c)=>{`+body+`}}}}}`)
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
			defer result.Release()
			require.NotEmpty(t, result.Errors)
			require.Empty(t, result.Results)
		})
	}
}
func TestJavaScriptRulesetTimeout(t *testing.T) {
	rs := loadJSRules(t, `export default {rules:{check:{given:'$',then:{function:()=>{while(true){}}}}}}`)
	started := time.Now()
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), Timeout: 50 * time.Millisecond, SilenceLogs: true}, &ExecutionOptions{RunTimeout: time.Second})
	defer result.Release()
	require.NotEmpty(t, result.Errors)
	require.Less(t, time.Since(started), 2*time.Second)
}
func TestJavaScriptRulesetFormats(t *testing.T) {
	rs := loadJSRules(t, `import {oas2} from '@stoplight/spectral-formats';export default {formats:[oas2],rules:{check:{given:'$',then:{function:()=>[{message:'must not run'}]}}}}`)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Empty(t, result.Results)
}
func TestJavaScriptRulesetNativeFunction(t *testing.T) {
	rs := loadJSRules(t, `import {truthy} from '@stoplight/spectral-functions';export default {rules:{check:{given:'$.info',then:{field:'title',function:truthy}}}}`)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {version: '1'}\npaths: {}"), SilenceLogs: true})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.True(t, strings.Contains(result.Results[0].Path, "title"))
}

func TestJavaScriptRulesetPositiveFormatAndPromise(t *testing.T) {
	rs := loadJSRules(t, `import {oas3} from '@stoplight/spectral-formats';import {createRulesetFunction} from '@stoplight/spectral-core';const f=createRulesetFunction({input:{type:'object'},options:null},async(v,o,c)=>[{message:'executed',path:[...c.path,'title']}]);export default {formats:[oas3],rules:{check:{given:'$.info',then:{function:f}}}}`)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "executed", result.Results[0].Message)
}
func TestJavaScriptRulesetConcurrentIsolation(t *testing.T) {
	rs := loadJSRules(t, `let calls=0;export default {rules:{check:{given:'$.info',then:{function:()=>[{message:String(++calls)}]}}}}`)
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
			defer result.Release()
			require.Empty(t, result.Errors)
			require.Len(t, result.Results, 1)
			require.Equal(t, "1", result.Results[0].Message)
		})
	}
}
func TestJavaScriptRulesetNativeMissingOptions(t *testing.T) {
	rs := loadJSRules(t, `import {length} from '@stoplight/spectral-functions';export default {rules:{check:{given:'$.info.title',then:{function:length}}}}`)
	require.NotEmpty(t, ValidateRuleSet(rs, nil))
}
func TestJavaScriptRulesetOptionsAreIsolated(t *testing.T) {
	rs := loadJSRules(t, `export default {rules:{check:{given:'$.info',then:{functionOptions:{count:0},function:(v,o)=>[{message:String(++o.count)}]}}}}`)
	for i := 0; i < 2; i++ {
		result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Hello, version: '1'}\npaths: {}"), SilenceLogs: true})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 1)
		require.Equal(t, "1", result.Results[0].Message)
		result.Release()
	}
}

func TestJavaScriptWrappedAlphabeticalPreservesOrder(t *testing.T) {
	rs := loadJSRules(t, `import {alphabetical} from '@stoplight/spectral-functions';export default {rules:{check:{given:'$.info',then:{function:(v,o,c)=>alphabetical(v,o,c)}}}}`)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {version: '1', title: Test}\npaths: {}"), SilenceLogs: true})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Results)
}

func TestJavaScriptFetchCancellation(t *testing.T) {
	cancelled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done(); close(cancelled) }))
	defer server.Close()
	rs := loadJSRules(t, `export default {rules:{check:{given:'$',then:{function:async()=>{await fetch('`+server.URL+`');return []}}}}}`)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}"), SilenceLogs: true, FetchConfig: &config.FetchConfig{AllowHTTP: true, AllowPrivateNetworks: true}}, &ExecutionOptions{RunTimeout: 100 * time.Millisecond})
	defer result.Release()
	require.NotEmpty(t, result.Errors)
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("fetch survived execution cancellation")
	}
}

func TestJavaScriptInheritedFormatIsolation(t *testing.T) {
	rs := loadJSRules(t, `const old={formats:[d=>d.swagger==='2.0'],rules:{old:{given:'$',then:{function:()=>[{message:'wrong format'}]}}}};const current={formats:[d=>d.openapi==='3.0.3'],rules:{current:{given:'$',then:{function:()=>[{message:'current'}]}}}};export default {extends:[old,current]}`)
	result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}"), SilenceLogs: true})
	defer result.Release()
	require.Empty(t, result.Errors)
	require.Len(t, result.Results, 1)
	require.Equal(t, "current", result.Results[0].Message)
}

func TestJavaScriptInheritedUnrestrictedFormats(t *testing.T) {
	for _, parents := range []string{`extends:[any,old]`, `extends:[any],formats:[d=>d.swagger==='2.0'],rules:{child:{given:'$',then:{function:()=>[{message:'child'}]}}}`} {
		rs := loadJSRules(t, `const any={rules:{any:{given:'$',then:{function:()=>[{message:'any'}]}}}};const old={formats:[d=>d.swagger==='2.0'],rules:{old:{given:'$',then:{function:()=>[{message:'old'}]}}}};export default {`+parents+`}`)
		result := ApplyRulesToRuleSet(&RuleSetExecution{RuleSet: rs, Spec: []byte("openapi: 3.0.3\ninfo: {title: Test, version: '1'}\npaths: {}"), SilenceLogs: true})
		require.Empty(t, result.Errors)
		require.Len(t, result.Results, 1)
		require.Equal(t, "any", result.Results[0].Message)
		result.Release()
	}
}
