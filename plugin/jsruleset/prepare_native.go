//go:build !js || !wasm

// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package jsruleset

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
)

// PrepareWithReader compiles modules using the supplied reader for every captured file.
func PrepareWithReader(ctx context.Context, entry string, readFile func(string) ([]byte, error)) (*Prepared, error) {
	return prepareWithSource(ctx, entry, readFile, nil)
}

func prepareWithSource(ctx context.Context, entry string, readFile func(string) ([]byte, error), source *string) (*Prepared, error) {
	if readFile == nil {
		readFile = os.ReadFile
	}
	entry, err := filepath.Abs(entry)
	if err != nil {
		return nil, err
	}
	captured := map[string]string{}
	var mu sync.Mutex
	capture := func(path string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		b, e := readFile(path)
		if e != nil {
			return nil, e
		}
		hash := fmt.Sprintf("%x", sha256.Sum256(b))
		if old, ok := captured[path]; ok && old != hash {
			return nil, fmt.Errorf("module changed during preparation: %s", path)
		}
		captured[path] = hash
		return b, nil
	}
	plugin := api.Plugin{Name: "vacuum-modules", Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(a api.OnResolveArgs) (api.OnResolveResult, error) {
			if err := ctx.Err(); err != nil {
				return api.OnResolveResult{}, err
			}
			if a.Kind == api.ResolveJSDynamicImport {
				return api.OnResolveResult{}, fmt.Errorf("dynamic import is unsupported in %s", a.Importer)
			}
			if source, ok := compatibilityModule(a.Path); ok {
				return api.OnResolveResult{Path: a.Path, Namespace: "vacuum", PluginData: source}, nil
			}
			if strings.HasPrefix(a.Path, "@stoplight/") || strings.HasPrefix(a.Path, "@openlint/") || (strings.Contains(a.Path, ":") && !filepath.IsAbs(a.Path)) {
				return api.OnResolveResult{}, fmt.Errorf("unsupported module %q imported by %s", a.Path, a.Importer)
			}
			return api.OnResolveResult{}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "vacuum"}, func(a api.OnLoadArgs) (api.OnLoadResult, error) {
			s := a.PluginData.(string)
			return api.OnLoadResult{Contents: &s, Loader: api.LoaderJS}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "file"}, func(a api.OnLoadArgs) (api.OnLoadResult, error) {
			if !IsModule(a.Path) && filepath.Ext(a.Path) != ".json" {
				return api.OnLoadResult{}, fmt.Errorf("unsupported module %s; compile TypeScript to JavaScript first", a.Path)
			}
			b, e := capture(a.Path)
			if e != nil {
				return api.OnLoadResult{}, e
			}
			for dir := filepath.Dir(a.Path); ; dir = filepath.Dir(dir) {
				metadata := filepath.Join(dir, "package.json")
				if _, e := os.Stat(metadata); e == nil {
					if _, e = capture(metadata); e != nil {
						return api.OnLoadResult{}, e
					}
				}
				if filepath.Dir(dir) == dir {
					break
				}
			}
			s := string(b)
			loader := api.LoaderJS
			if filepath.Ext(a.Path) == ".json" {
				loader = api.LoaderJSON
			}
			return api.OnLoadResult{Contents: &s, Loader: loader, ResolveDir: filepath.Dir(a.Path)}, nil
		})
	}}
	options := api.BuildOptions{EntryPoints: []string{entry}, Bundle: true, PreserveSymlinks: true, Write: false, Format: api.FormatCommonJS, Platform: api.PlatformNeutral, MainFields: []string{"module", "main"}, Conditions: []string{"default"}, Target: api.ES2017, Plugins: []api.Plugin{plugin}, LogLevel: api.LogLevelSilent, TreeShaking: api.TreeShakingFalse, Metafile: true}
	if source != nil {
		options.EntryPoints = nil
		options.Stdin = &api.StdinOptions{Contents: *source, ResolveDir: filepath.Dir(entry), Sourcefile: entry, Loader: api.LoaderJS}
	}
	result := api.Build(options)
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("JavaScript ruleset %s: %s", entry, strings.Join(api.FormatMessages(result.Errors, api.FormatMessagesOptions{Kind: api.ErrorMessage}), "\n"))
	}
	if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("ruleset must compile to one module")
	}
	code := string(result.OutputFiles[0].Contents)
	if strings.Contains(code, `throw Error('Dynamic require of "'`) {
		return nil, fmt.Errorf("dynamic require is unsupported in ruleset %s", entry)
	}
	program, err := goja.Compile(entry, code, false)
	if err != nil {
		return nil, err
	}
	p := &Prepared{program: program, entry: entry}
	scope, err := p.open(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer scope.Close()
	p.normalized = scope.normalized
	for name := range scope.functions {
		p.names = append(p.names, name)
	}
	sort.Strings(p.names)
	for path, hash := range captured {
		p.files = append(p.files, File{path, hash})
	}
	sort.Slice(p.files, func(i, j int) bool { return p.files[i].Path < p.files[j].Path })
	return p, nil
}

// PrepareFunctions binds named local functions through the existing module runtime.
// The synthetic module contains function registrations only; declarative rules
// keep their original loader semantics.
func PrepareFunctions(ctx context.Context, source string, files map[string]string, readFile func(string) ([]byte, error)) (*Prepared, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var module strings.Builder
	module.WriteString("const rules={};\n")
	for i, name := range names {
		path, _ := json.Marshal(files[name])
		key, _ := json.Marshal(name)
		fmt.Fprintf(&module, "import f%d from %s; rules[%s]={given:'$',then:{function:f%d}};\n", i, path, key, i)
	}
	module.WriteString("export default {rules};\n")
	text := module.String()
	prepared, err := prepareWithSource(ctx, source, readFile, &text)
	if err != nil {
		return nil, err
	}
	var normalized struct {
		Rules map[string]struct {
			Then struct {
				Function string `json:"function"`
			} `json:"then"`
		} `json:"rules"`
	}
	if err = json.Unmarshal(prepared.Data(), &normalized); err != nil {
		return nil, err
	}
	prepared.functionNames = make(map[string]string, len(names))
	for _, name := range names {
		prepared.functionNames[name] = normalized.Rules[name].Then.Function
	}
	return prepared, nil
}
