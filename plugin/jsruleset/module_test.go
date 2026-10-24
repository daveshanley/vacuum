package jsruleset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pb33f/testify/require"
)

func writeModule(t *testing.T, dir, name, code string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0700))
	require.NoError(t, os.WriteFile(p, []byte(code), 0600))
	return p
}
func TestPrepareModuleGraphs(t *testing.T) {
	for _, extension := range []string{"js", "cjs", "mjs"} {
		t.Run(extension, func(t *testing.T) {
			dir := t.TempDir()
			writeModule(t, dir, "helper.cjs", `module.exports=v=>v?undefined:[{message:'missing'}]`)
			writeModule(t, dir, "settings.json", `{"severity":"error"}`)
			code := `const fn=require('./helper.cjs'), settings=require('./settings.json');module.exports={rules:{check:{given:'$',severity:settings.severity,then:{function:fn}}}}`
			if extension == "mjs" {
				code = `import fn from './helper.cjs';import settings from './settings.json';export default {rules:{check:{given:'$',severity:settings.severity,then:{function:fn}}}}`
			}
			entry := writeModule(t, dir, "rules."+extension, code)
			p, err := Prepare(context.Background(), entry)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(p.Files()), 3)
			require.Len(t, p.Functions(), 1)
			scope, err := p.Open(context.Background(), nil)
			require.NoError(t, err)
			scope.Close()
			scope.Close()
		})
	}
}
func TestPrepareRejectsUnsupportedModules(t *testing.T) {
	for name, code := range map[string]string{
		"factory": `export default ()=>({rules:{}})`, "promise": `export default Promise.resolve({rules:{}})`,
		"builtin": `import fs from 'node:fs';export default {rules:fs.readFileSync('x')}`,
		"dynamic": `const p='./helper.js';export default require(p)`, "dynamic-import": `export default {rules:{x:{given:'$',then:{function:()=>import('./helper.js')}}}}`,
		"bad-export": `export default {rules:{},resolver:{}}`, "unknown-helper": `import {missing} from '@stoplight/spectral-functions';export default {rules:{x:{given:'$',then:{function:missing}}}}`,
		"top-level-await": `await Promise.resolve();export default {rules:{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			entry := writeModule(t, t.TempDir(), "rules.mjs", code)
			_, err := Prepare(context.Background(), entry)
			require.Error(t, err)
		})
	}
}
func TestPrepareCancellation(t *testing.T) {
	entry := writeModule(t, t.TempDir(), "rules.cjs", `while(true){}`)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := Prepare(ctx, entry)
	require.Error(t, err)
	require.Less(t, time.Since(started), time.Second)
}

func TestPrepareRejectsUnsupportedCompatibilityImports(t *testing.T) {
	for _, prefix := range []string{"@stoplight/spectral-", "@openlint/openlint-"} {
		for _, tc := range []struct{ module, export string }{
			{"functions", "unreferencedReusableObject"},
			{"core", "Document"},
			{"formats", "missing"},
			{"runtime", "fetch"},
		} {
			t.Run(prefix+tc.module+"/"+tc.export, func(t *testing.T) {
				entry := writeModule(t, t.TempDir(), "rules.mjs", fmt.Sprintf(`import {%s} from '%s%s';export default {rules:{check:{given:'$',then:{function:%s}}}};`, tc.export, prefix, tc.module, tc.export))
				_, err := Prepare(context.Background(), entry)
				require.ErrorContains(t, err, tc.module)
			})
		}
	}
}
func TestCreateRulesetFunctionValidatesOptions(t *testing.T) {
	entry := writeModule(t, t.TempDir(), "rules.mjs", `import {createRulesetFunction} from '@stoplight/spectral-core';const check=createRulesetFunction({input:null,options:{type:'object',required:['prefix']}},()=>[]);export default {rules:{x:{given:'$',then:{function:check,functionOptions:{}}}}}`)
	_, err := Prepare(context.Background(), entry)
	require.ErrorContains(t, err, "prefix")
}

func TestPrepareUsesReader(t *testing.T) {
	entry := writeModule(t, t.TempDir(), "rules.cjs", `throw Error('disk must not run')`)
	supplied := []byte(`module.exports={rules:{x:{given:'$',then:{function:()=>[]}}}}`)
	p, err := PrepareWithReader(context.Background(), entry, func(path string) ([]byte, error) {
		if path == entry {
			return supplied, nil
		}
		return os.ReadFile(path)
	})
	require.NoError(t, err)
	require.NotNil(t, p)
	_, err = PrepareWithReader(context.Background(), entry, func(string) ([]byte, error) { return nil, os.ErrPermission })
	require.Error(t, err)
}
func TestPrimitiveInheritedOverrides(t *testing.T) {
	entry := writeModule(t, t.TempDir(), "rules.mjs", `const parent={extends:['spectral:oas'],rules:{'info-contact':false,'info-description':'error'}};export default {extends:[parent]}`)
	p, err := Prepare(context.Background(), entry)
	require.NoError(t, err)
	require.Contains(t, string(p.Data()), `"info-contact":false`)
	require.Contains(t, string(p.Data()), `"info-description":"error"`)
}

func TestModuleReaderCallsAreSerialized(t *testing.T) {
	dir := t.TempDir()
	var code strings.Builder
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("helper%d.mjs", i)
		writeModule(t, dir, name, `export default ()=>[]`)
		fmt.Fprintf(&code, "import f%d from './%s';\n", i, name)
	}
	code.WriteString("export default {rules:{x:{given:'$',then:{function:f0}}}}")
	entry := writeModule(t, dir, "rules.mjs", code.String())
	var active atomic.Int32
	var concurrent atomic.Bool
	_, err := PrepareWithReader(context.Background(), entry, func(path string) ([]byte, error) {
		if active.Add(1) > 1 {
			concurrent.Store(true)
		}
		defer active.Add(-1)
		time.Sleep(time.Millisecond)
		return os.ReadFile(path)
	})
	require.NoError(t, err)
	require.False(t, concurrent.Load())
}
