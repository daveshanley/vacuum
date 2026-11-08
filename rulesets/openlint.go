// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package rulesets

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/plugin/jsruleset"
	"github.com/dlclark/regexp2"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonschema/v6"
	"io"
	"maps"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
)

// The unmodified schema is from OpenLint spec revision
// 4e95c96e4a7ecb17320cd11fc2fcd2571e8d0291; see schemas/OPENLINT-LICENSE.
//
//go:embed schemas/openlint-v1.schema.json
var openLintSchema string

type openLintDefinition struct {
	Overrides     []map[string]any `json:"overrides"`
	ParserOptions map[string]any   `json:"parserOptions"`
	Functions     []string         `json:"functions"`
	FunctionsDir  string           `json:"functionsDir"`
	client        *http.Client
	pointers      []openLintPointerOverride
	scopes        []*RuleSet
}

type openLintRule struct {
	properties map[string]any
	enabled    bool
	owner      *RuleSet
}

func createOpenLintRuleSet(data []byte) (*RuleSet, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	schema, err := compiledOpenLintSchema()
	if err != nil {
		return nil, err
	}
	if err = schema.Validate(raw); err != nil {
		return nil, fmt.Errorf("invalid OpenLint ruleset: %w", err)
	}
	rs := &RuleSet{Contract: model.OpenLintV1Draft, openLint: &openLintDefinition{}}
	if err = json.Unmarshal(data, rs); err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, rs.openLint); err != nil {
		return nil, err
	}
	if len(rs.Aliases) > 0 {
		rs.ParsedAliases, err = ParseAliases(rs.Aliases)
		if err != nil {
			return nil, err
		}
		resolved := ResolveAliasesWithMatcher(rs.ParsedAliases, "", func(string, string) bool { return true })
		if _, err = ExpandAliasReferences(resolved); err != nil {
			return nil, err
		}
		for key := range resolved {
			if _, err = ExpandRuleGivenPaths([]string{"#" + key}, resolved); err != nil {
				return nil, err
			}
		}
	}
	return rs, nil
}

func (rsm ruleSetsModel) generateOpenLintRuleSet(root *RuleSet, client *http.Client) *RuleSet {
	return rsm.generateOpenLintRuleSetWithContext(context.Background(), root, client)
}

func (rsm ruleSetsModel) generateOpenLintRuleSetWithContext(parent context.Context, root *RuleSet, client *http.Client) *RuleSet {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	active := map[string]bool{}
	var walk func(*RuleSet) error
	walk = func(rs *RuleSet) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := rs.validateOpenLintFunctionSource(); err != nil {
			return err
		}
		location := rs.sourceLocation.path
		if rs.sourceLocation.remoteURL != nil {
			location = rs.sourceLocation.remoteURL.String()
		}
		if active[location] {
			return fmt.Errorf("ruleset inheritance cycle at %s", location)
		}
		active[location] = true
		defer delete(active, location)
		entries := []any{}
		switch e := rs.Extends.(type) {
		case string:
			entries = append(entries, e)
		case []any:
			entries = e
		}
		for _, entry := range entries {
			mode := "recommended"
			name, _ := entry.(string)
			if tuple, ok := entry.([]any); ok {
				name, _ = tuple[0].(string)
				mode, _ = tuple[1].(string)
			}
			var parent *RuleSet
			var err error
			if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
				parent, err = DownloadRemoteRuleSet(ctx, name, client, model.OpenLintV1Draft)
			} else if rs.sourceLocation.remoteURL != nil {
				ref, e := rs.sourceLocation.remoteURL.Parse(name)
				if e != nil {
					return e
				}
				parent, err = DownloadRemoteRuleSet(ctx, ref.String(), client, model.OpenLintV1Draft)
			} else {
				name = filepath.Join(filepath.Dir(location), name)
				parent, err = LoadLocalRuleSetWithReader(ctx, name, rs.readLocalFile, model.OpenLintV1Draft)
			}
			if err != nil {
				return err
			}
			if err = walk(parent); err != nil {
				return err
			}
			rs.openLintParents = append(rs.openLintParents, parent)
			rs.openLintParentModes = append(rs.openLintParentModes, mode)
		}
		rs.openLint.client = client
		return rs.compileOpenLintRules(nil)
	}
	if err := walk(root); err != nil {
		root.addLoadError(err)
	}
	return root
}

func (rs *RuleSet) compileOpenLintRules(matches map[string]bool) error {
	definitions := map[string]openLintRule{}
	for i, parent := range rs.openLintParents {
		for name, rule := range parent.openLintRules {
			rule.properties = maps.Clone(rule.properties)
			switch rs.openLintParentModes[i] {
			case "off":
				rule.enabled = false
			case "all":
				rule.enabled = true
			case "recommended":
				if recommended, ok := rule.properties["recommended"].(bool); ok && !recommended {
					rule.enabled = false
				}
			}
			if previous, exists := definitions[name]; exists {
				for k, v := range rule.properties {
					previous.properties[k] = v
				}
				previous.enabled = rule.enabled
				definitions[name] = previous
			} else {
				definitions[name] = rule
			}
		}
	}
	if err := applyOpenLintDefinitions(definitions, rs.RuleDefinitions, rs); err != nil {
		return err
	}

	rs.openLintRules = definitions
	return rs.renderOpenLintRules(matches, false)
}

func (rs *RuleSet) renderOpenLintRules(matches map[string]bool, includeDisabled bool) error {
	definitions := rs.openLintRules
	rs.FormatMatches = maps.Clone(matches)
	rs.Rules = map[string]*model.Rule{}
	for name, definition := range definitions {
		properties := maps.Clone(definition.properties)
		if properties["given"] == nil || properties["then"] == nil {
			return fmt.Errorf("rule %q requires given and then", name)
		}
		severity := "warn"
		switch v := properties["severity"].(type) {
		case string:
			severity = v
		case float64:
			severity = map[int]string{-1: "off", 0: "error", 1: "warn", 2: "info", 3: "hint"}[int(v)]
		}
		properties["severity"] = severity
		if _, ok := properties["resolved"]; !ok {
			properties["resolved"] = true
		}
		if _, ok := properties["formats"]; !ok {
			properties["formats"] = definition.owner.Formats
		}
		if _, ok := properties["documentationUrl"]; !ok && definition.owner.DocumentationURI != "" {
			properties["documentationUrl"] = definition.owner.DocumentationURI + "#" + name
		}
		data, err := json.Marshal(properties)
		if err != nil {
			return err
		}
		var rule model.Rule
		if err = json.Unmarshal(data, &rule); err != nil {
			return err
		}
		rule.Id = name
		rule.RuleCategory = model.RuleCategories[model.CategoryValidation]
		aliases, err := openLintAliases(definition.owner.ParsedAliases, matches)
		if err != nil {
			return err
		}
		given, err := ExpandRuleGivenPaths(openLintGiven(rule.Given), aliases)
		if err != nil {
			return err
		}
		rule.Given = given
		rule.Then = bindOpenLintActions(rule.Then, definition.owner)
		if includeDisabled || definition.enabled && severity != "off" && len(given) > 0 {
			rs.Rules[name] = &rule
		}
	}
	return nil
}

// RulesForValidation returns all effective definitions, including disabled rules.
// OpenLint validates definitions before activation; the execution set is unchanged.
func (rs *RuleSet) RulesForValidation() (map[string]*model.Rule, error) {
	if rs == nil {
		return nil, nil
	}
	if rs.Contract != model.OpenLintV1Draft || rs.openLint == nil {
		return rs.Rules, nil
	}
	view := rs.CopyForExecution()
	if err := view.renderOpenLintRules(nil, true); err != nil {
		return nil, err
	}
	return view.Rules, nil
}

func (rs *RuleSet) validateOpenLintFunctionSource() error {
	if rs.sourceLocation.remoteURL != nil && rs.openLint != nil && len(rs.openLint.Functions) > 0 {
		return fmt.Errorf("remote ruleset %q cannot load local JavaScript functions; declare them in a local ruleset", rs.sourceLocation.remoteURL.String())
	}
	return nil
}

func openLintGiven(value any) []string {
	switch v := value.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, p := range v {
			if s, ok := p.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

var openLintSchemaOnce sync.Once
var openLintSchemaCompiled *jsonschema.Schema
var openLintSchemaError error

type openLintSchemaRegexp struct{ *regexp2.Regexp }

func (r openLintSchemaRegexp) MatchString(value string) bool {
	matched, err := r.Regexp.MatchString(value)
	return err == nil && matched
}

func compiledOpenLintSchema() (*jsonschema.Schema, error) {
	openLintSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.UseRegexpEngine(func(pattern string) (jsonschema.Regexp, error) {
			rx, err := regexp2.Compile(pattern, regexp2.ECMAScript)
			if err != nil {
				return nil, err
			}
			rx.MatchTimeout = time.Second
			return openLintSchemaRegexp{rx}, nil
		})
		var data any
		if openLintSchemaError = json.Unmarshal([]byte(openLintSchema), &data); openLintSchemaError != nil {
			return
		}
		if openLintSchemaError = compiler.AddResource("openlint-schema.json", data); openLintSchemaError != nil {
			return
		}
		openLintSchemaCompiled, openLintSchemaError = compiler.Compile("openlint-schema.json")
	})
	return openLintSchemaCompiled, openLintSchemaError
}

func applyOpenLintDefinitions(definitions map[string]openLintRule, values map[string]any, owner *RuleSet) error {
	for name, value := range values {
		previous, exists := definitions[name]
		if properties, ok := value.(map[string]any); ok {
			if !exists {
				previous = openLintRule{properties: map[string]any{}, owner: owner}
			}
			previous.properties = maps.Clone(previous.properties)
			for key, value := range properties {
				previous.properties[key] = value
			}
			recommended, exists := previous.properties["recommended"].(bool)
			previous.enabled = !exists || recommended
		} else {
			if !exists {
				return fmt.Errorf("rule %q is not defined", name)
			}
			previous.properties = maps.Clone(previous.properties)
			switch v := value.(type) {
			case bool:
				previous.enabled = v
			case string, float64:
				previous.properties["severity"] = v
				previous.enabled = true
			}
		}
		definitions[name] = previous
	}
	return nil
}

func openLintAliases(parsed map[string]*ParsedAlias, matches map[string]bool) (map[string][]string, error) {
	aliases := map[string][]string{}
	for name, alias := range parsed {
		if alias.Simple != nil {
			aliases[name] = []string(alias.Simple)
			continue
		}
		aliases[name] = nil
		if alias.Targeted != nil {
			for _, target := range alias.Targeted.Targets {
				for _, format := range target.Formats {
					if matches == nil || matches[format] {
						aliases[name] = target.Given
						break
					}
				}
			}
		}
	}
	return ExpandAliasReferences(aliases)
}

// OpenLintFormats returns the format identifiers declared by this ruleset and
// its parents. Alias-only identifiers do not participate in detection.
func (rs *RuleSet) OpenLintFormats() []string {
	names := map[string]bool{}
	var walk func(*RuleSet)
	walk = func(source *RuleSet) {
		if source == nil {
			return
		}
		for _, name := range source.Formats {
			names[name] = true
		}
		for _, value := range source.RuleDefinitions {
			if rule, ok := value.(map[string]any); ok {
				for _, name := range openLintGiven(rule["formats"]) {
					names[name] = true
				}
			}
		}
		for _, parent := range source.openLintParents {
			walk(parent)
		}
		if source.openLint != nil {
			for _, scope := range source.openLint.scopes {
				walk(scope)
			}
		}
	}
	walk(rs)
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type openLintPointerOverride struct {
	path  []string
	rules map[string]any
}

// ForOpenLintDocument prepares an isolated rule set for a source document.
// Loading-time definitions remain immutable across concurrent invocations.
func (rs *RuleSet) ForOpenLintDocument(ctx context.Context, source string, detect func([]string) map[string]bool) (*RuleSet, error) {
	if rs == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := rs.validateOpenLintFunctionSource(); err != nil {
		return nil, err
	}
	if rs.openLint == nil {
		result := rs.CopyForExecution()
		result.Contract = model.OpenLintV1Draft
		if detect != nil {
			result.FormatMatches = detect(result.OpenLintFormats())
		}
		return result, nil
	}
	result := rs.CopyForExecution()
	definition := *rs.openLint
	definition.pointers = nil
	definition.scopes = nil
	definition.ParserOptions = maps.Clone(rs.openLint.ParserOptions)
	result.openLint = &definition
	result.openLintParents = nil
	for _, parent := range rs.openLintParents {
		prepared, err := parent.ForOpenLintDocument(ctx, source, nil)
		if err != nil {
			return nil, err
		}
		result.openLintParents = append(result.openLintParents, prepared)
		result.openLint.pointers = append(result.openLint.pointers, prepared.openLint.pointers...)
	}
	if err := result.compileOpenLintRules(nil); err != nil {
		return nil, err
	}
	for _, override := range rs.openLint.Overrides {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pointers, match, err := rs.matchOpenLintOverride(source, override)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}
		if pointers != nil {
			values, _ := override["rules"].(map[string]any)
			for _, pointer := range pointers {
				result.openLint.pointers = append(result.openLint.pointers, openLintPointerOverride{pointer, values})
			}
			continue
		}
		raw := maps.Clone(override)
		delete(raw, "files")
		if _, ok := raw["rules"]; !ok {
			raw["rules"] = map[string]any{}
		}
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		fragment, err := createOpenLintRuleSet(data)
		if err != nil {
			return nil, err
		}
		fragment.sourceLocation = rs.sourceLocation
		fragment.readLocalFile = rs.readLocalFile
		aliases := maps.Clone(rs.Aliases)
		if aliases == nil {
			aliases = map[string]any{}
		}
		for name, value := range fragment.Aliases {
			aliases[name] = value
		}
		fragment.Aliases = aliases
		fragment.ParsedAliases, err = ParseAliases(aliases)
		if err != nil {
			return nil, err
		}
		if _, has := override["formats"]; !has {
			fragment.Formats = rs.Formats
		}
		if fragment.DocumentationURI == "" {
			fragment.DocumentationURI = rs.DocumentationURI
		}
		values := fragment.RuleDefinitions
		fragment.RuleDefinitions = nil
		model := ruleSetsModel{}
		model.generateOpenLintRuleSetWithContext(ctx, fragment, rs.openLint.client)
		if err := fragment.LoadError(); err != nil {
			return nil, err
		}
		fragment.RuleDefinitions = values
		result.openLint.scopes = append(result.openLint.scopes, fragment)
		if len(fragment.openLint.ParserOptions) > 0 {
			if result.openLint.ParserOptions == nil {
				result.openLint.ParserOptions = map[string]any{}
			}
			for key, value := range fragment.openLint.ParserOptions {
				result.openLint.ParserOptions[key] = value
			}
		}
		if _, hasExtends := override["extends"]; hasExtends {
			if _, hasRules := override["rules"]; !hasRules {
				for name, rule := range result.openLintRules {
					if _, own := rs.RuleDefinitions[name]; !own && rule.owner != result {
						delete(result.openLintRules, name)
					}
				}
			}
			for name, rule := range fragment.openLintRules {
				result.openLintRules[name] = rule
			}
		}
		if err = applyOpenLintDefinitions(result.openLintRules, values, fragment); err != nil {
			return nil, err
		}
	}
	var matches map[string]bool
	if detect != nil {
		matches = detect(result.OpenLintFormats())
	}
	if err := result.renderOpenLintRules(matches, false); err != nil {
		return nil, err
	}
	if err := result.prepareOpenLintFunctions(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (rs *RuleSet) matchOpenLintOverride(source string, override map[string]any) ([][]string, bool, error) {
	if source == "" {
		return nil, false, nil
	}
	directory := filepath.Dir(rs.sourceLocation.path)
	if !filepath.IsAbs(directory) {
		var err error
		directory, err = filepath.Abs(directory)
		if err != nil {
			return nil, false, err
		}
	}
	relative, err := filepath.Rel(directory, source)
	if err != nil {
		return nil, false, err
	}
	relative = filepath.ToSlash(relative)
	patterns := openLintGiven(override["files"])
	// Scope depends on every entry, including globs that do not match this file.
	pointerScoped := true
	for _, pattern := range patterns {
		if !strings.Contains(pattern, "#") {
			pointerScoped = false
			break
		}
	}
	var pointers [][]string
	matched := false
	for _, pattern := range patterns {
		glob, pointer, hasPointer := strings.Cut(pattern, "#")
		ok, err := openLintGlob(glob, relative)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			continue
		}
		matched = true
		if pointerScoped && hasPointer {
			segments := []string{}
			if pointer != "" {
				for _, segment := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
					segments = append(segments, strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~"))
				}
			}
			pointers = append(pointers, segments)
		}
	}
	return pointers, matched, nil
}

func openLintGlob(pattern, name string) (bool, error) {
	parts, segments := strings.Split(pattern, "/"), strings.Split(name, "/")
	type position struct{ p, n int }
	memo := map[position]bool{}
	seen := map[position]bool{}
	var visit func(int, int) (bool, error)
	visit = func(p, n int) (bool, error) {
		key := position{p, n}
		if seen[key] {
			return memo[key], nil
		}
		seen[key] = true
		if p == len(parts) {
			return n == len(segments), nil
		}
		if parts[p] == "**" {
			ok, err := visit(p+1, n)
			if err != nil || ok {
				memo[key] = ok
				return ok, err
			}
			if n < len(segments) {
				ok, err = visit(p, n+1)
				memo[key] = ok
				return ok, err
			}
			return false, nil
		}
		if n >= len(segments) {
			return false, nil
		}
		ok, err := path.Match(parts[p], segments[n])
		if err != nil || !ok {
			return false, err
		}
		ok, err = visit(p+1, n+1)
		memo[key] = ok
		return ok, err
	}
	return visit(0, 0)
}

// OpenLintSeverity applies pointer-scoped severity overrides to a result path.
// The returned value "off" means that the result is disabled at this location.
func (rs *RuleSet) OpenLintSeverity(ruleID string, resultPath []string, current string) string {
	if rs == nil || rs.openLint == nil {
		return current
	}
	for _, override := range rs.openLint.pointers {
		if len(override.path) > len(resultPath) {
			continue
		}
		match := true
		for i, segment := range override.path {
			if segment != resultPath[i] {
				match = false
				break
			}
		}
		if match {
			if value, ok := override.rules[ruleID]; ok {
				current = openLintSeverity(value, current)
			}
		}
	}
	return current
}

func openLintSeverity(value any, fallback string) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return map[int]string{-1: "off", 0: "error", 1: "warn", 2: "info", 3: "hint"}[int(v)]
	}
	return fallback
}

// ParserSeverity returns the configured parser diagnostic severity.
func (rs *RuleSet) ParserSeverity(kind, fallback string) string {
	if rs == nil || rs.openLint == nil {
		return fallback
	}
	return openLintSeverity(rs.openLint.ParserOptions[kind], fallback)
}

func functionBinding(owner *RuleSet, name string) string {
	directory := owner.openLint.FunctionsDir
	if directory == "" {
		directory = "functions"
	}
	return filepath.Join(filepath.Dir(owner.sourceLocation.path), directory, name+".js")
}
func bindOpenLintActions(value any, owner *RuleSet) any {
	if owner == nil || owner.openLint == nil || len(owner.openLint.Functions) == 0 {
		return value
	}
	switch actions := value.(type) {
	case []any:
		copy := make([]any, len(actions))
		for i, action := range actions {
			copy[i] = bindOpenLintActions(action, owner)
		}
		return copy
	case map[string]any:
		copy := maps.Clone(actions)
		name, _ := copy["function"].(string)
		for _, custom := range owner.openLint.Functions {
			if name == custom {
				copy["function"] = functionBinding(owner, name)
				break
			}
		}
		return copy
	}
	return value
}

func (rs *RuleSet) prepareOpenLintFunctions(parent context.Context) error {
	files := map[string]string{}
	owners := map[*RuleSet]bool{}
	var collect func(*RuleSet) error
	collect = func(owner *RuleSet) error {
		if err := parent.Err(); err != nil {
			return err
		}
		if owner == nil || owners[owner] {
			return nil
		}
		if err := owner.validateOpenLintFunctionSource(); err != nil {
			return err
		}
		owners[owner] = true
		if owner.openLint != nil {
			directory := owner.openLint.FunctionsDir
			if directory == "" {
				directory = "functions"
			}
			for _, name := range owner.openLint.Functions {
				files[functionBinding(owner, name)] = filepath.Join(filepath.Dir(owner.sourceLocation.path), directory, name+".js")
			}
		}
		for _, inherited := range owner.openLintParents {
			if err := collect(inherited); err != nil {
				return err
			}
		}
		if owner.openLint != nil {
			for _, scope := range owner.openLint.scopes {
				if err := collect(scope); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := collect(rs); err != nil {
		return err
	}
	for _, rule := range rs.openLintRules {
		if err := collect(rule.owner); err != nil {
			return err
		}
	}
	if len(files) == 0 {
		return nil
	}
	read := rs.readLocalFile
	if read == nil {
		read = os.ReadFile
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	prepared, err := jsruleset.PrepareFunctions(ctx, rs.sourceLocation.path, files, read)
	if err != nil {
		return err
	}
	rs.Executable = prepared
	return nil
}

func validateOpenLintDocument(data []byte) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var first, second yaml.Node
	if err := decoder.Decode(&first); err != nil {
		return fmt.Errorf("invalid ruleset document: %w", err)
	}
	if err := decoder.Decode(&second); err != io.EOF {
		if err != nil {
			return fmt.Errorf("invalid ruleset document: %w", err)
		}
		return fmt.Errorf("ruleset must contain exactly one document")
	}
	return nil
}
