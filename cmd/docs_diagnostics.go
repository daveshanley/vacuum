// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	doctorV3 "github.com/pb33f/doctor/model/high/v3"
	ppmodel "github.com/pb33f/doctor/printingpress/model"
	"golang.org/x/sync/errgroup"
)

type docsDiagnosticsContext struct {
	enabled          bool
	flags            *LintFlags
	selectedRuleset  *rulesets.RuleSet
	customFunctions  map[string]model.RuleFunction
	ignoredItems     model.IgnoredItems
	httpClientConfig utils.HTTPClientConfig
	fetchConfig      *utils.FetchConfig
	fingerprint      string
}

type docsDiagnosticsProgressFunc func(completed, total int, currentSpec string, elapsed time.Duration)

func newDocsDiagnosticsContext(flags *LintFlags, httpClientConfig utils.HTTPClientConfig, fetchConfig *utils.FetchConfig, enabled bool) (*docsDiagnosticsContext, error) {
	ctx := &docsDiagnosticsContext{
		enabled:          enabled,
		flags:            flags,
		httpClientConfig: httpClientConfig,
		fetchConfig:      fetchConfig,
	}
	if !enabled {
		ctx.fingerprint = docsDiagnosticsFingerprint(false, flags, nil)
		return ctx, nil
	}

	selectedRS, err := LoadRulesetWithConfig(flags, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("unable to load diagnostics ruleset: %w", err)
	}
	customFunctions, err := LoadCustomFunctions(flags.FunctionsFlag, true)
	if err != nil {
		return nil, fmt.Errorf("unable to load custom functions: %w", err)
	}
	ignoredItems, err := LoadIgnoreFile(flags.IgnoreFile, true, true, true)
	if err != nil {
		return nil, err
	}

	ctx.selectedRuleset = selectedRS
	ctx.customFunctions = customFunctions
	ctx.ignoredItems = ignoredItems
	ctx.fingerprint = docsDiagnosticsFingerprint(true, flags, selectedRS)
	return ctx, nil
}

func (d *docsDiagnosticsContext) lintSpec(specBytes []byte, specPath string) ([]*doctorV3.RuleFunctionResult, error) {
	if d == nil || !d.enabled {
		return nil, nil
	}
	base, err := resolveDocsLintBase(specPath, d.flags)
	if err != nil {
		return nil, err
	}
	resultSet, execution, err := LintLoadedSpec(
		d.selectedRuleset,
		specBytes,
		d.customFunctions,
		base,
		d.flags.RemoteFlag,
		d.flags.SkipCheckFlag,
		time.Duration(d.flags.TimeoutFlag)*time.Second,
		time.Duration(d.flags.LookupTimeoutFlag)*time.Millisecond,
		d.httpClientConfig,
		d.fetchConfig,
		d.ignoredItems,
		&TurboFlags{TurboMode: d.flags.TurboMode},
		&ExecutionFlags{
			ResolveAllRefs:                  d.flags.ResolveAllRefs,
			NestedRefsDocContext:            d.flags.NestedRefsDocContext,
			SpecFilePath:                    specPath,
			ExtractReferencesFromExtensions: d.flags.ExtRefsFlag,
			IgnoreCircularArrayRef:          d.flags.IgnoreArrayCircleRef,
			IgnoreCircularPolymorphicRef:    d.flags.IgnorePolymorphCircleRef,
		},
	)
	if execution != nil {
		defer releaseDocsLintResources(execution)
	}
	if err != nil {
		return nil, err
	}
	if execution != nil && len(execution.Errors) > 0 {
		return nil, execution.Errors[0]
	}
	if resultSet == nil {
		return nil, nil
	}
	converted := make([]*doctorV3.RuleFunctionResult, 0, len(resultSet.Results))
	for _, result := range resultSet.Results {
		if result == nil {
			continue
		}
		converted = append(converted, doctorV3.ConvertRuleResult(result))
	}
	return converted, nil
}

func (d *docsDiagnosticsContext) lintCatalog(catalog *ppmodel.CatalogSite, report docsDiagnosticsProgressFunc) (map[string][]*doctorV3.RuleFunctionResult, error) {
	if d == nil || !d.enabled || catalog == nil {
		return nil, nil
	}
	jobs := docsCatalogLintJobs(catalog)
	results := make(map[string][]*doctorV3.RuleFunctionResult, len(jobs))
	if len(jobs) == 0 {
		return results, nil
	}
	start := time.Now()
	if report != nil {
		report(0, len(jobs), "", 0)
	}

	workerLimit := runtime.GOMAXPROCS(0)
	if workerLimit < 1 {
		workerLimit = 1
	}
	if workerLimit > len(jobs) {
		workerLimit = len(jobs)
	}

	var mu sync.Mutex
	completed := 0
	var group errgroup.Group
	group.SetLimit(workerLimit)
	for _, job := range jobs {
		job := job
		group.Go(func() error {
			specBytes, err := os.ReadFile(job.absPath)
			if err != nil {
				return fmt.Errorf("read catalog spec %s: %w", job.relativePath, err)
			}
			lintResults, err := d.lintSpec(specBytes, job.absPath)
			if err != nil {
				return fmt.Errorf("lint catalog spec %s: %w", job.relativePath, err)
			}
			mu.Lock()
			results[job.relativePath] = lintResults
			completed++
			currentCompleted := completed
			mu.Unlock()
			if report != nil {
				report(currentCompleted, len(jobs), job.relativePath, time.Since(start))
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

type docsCatalogLintJob struct {
	relativePath string
	absPath      string
}

func docsCatalogLintJobs(catalog *ppmodel.CatalogSite) []docsCatalogLintJob {
	if catalog == nil {
		return nil
	}
	var jobs []docsCatalogLintJob
	for _, service := range catalog.Services {
		if service == nil {
			continue
		}
		for _, version := range service.Versions {
			if version == nil {
				continue
			}
			for _, entry := range version.Entries {
				if entry == nil || strings.TrimSpace(entry.RelativePath) == "" {
					continue
				}
				jobs = append(jobs, docsCatalogLintJob{
					relativePath: entry.RelativePath,
					absPath:      filepath.Join(catalog.ScanRoot, filepath.FromSlash(entry.RelativePath)),
				})
			}
		}
	}
	return jobs
}

func resolveDocsLintBase(specPath string, flags *LintFlags) (string, error) {
	if flags != nil && strings.TrimSpace(flags.BaseFlag) != "" {
		if strings.Contains(flags.BaseFlag, "://") {
			return flags.BaseFlag, nil
		}
		abs, err := filepath.Abs(flags.BaseFlag)
		if err != nil {
			return "", fmt.Errorf("resolve lint base path: %w", err)
		}
		return abs, nil
	}
	if isDocsRemoteInput(specPath) {
		parsed, err := url.Parse(specPath)
		if err != nil {
			return "", fmt.Errorf("parse remote spec URL: %w", err)
		}
		dir := path.Dir(parsed.Path)
		if dir == "." {
			dir = "/"
		}
		if dir != "/" && !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		parsed.Path = dir
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String(), nil
	}
	return ResolveBasePathForFile(specPath, "")
}

func docsDiagnosticsFingerprint(enabled bool, flags *LintFlags, selectedRS *rulesets.RuleSet) string {
	payload := map[string]any{
		"diagnosticsEnabled": enabled,
		"vacuumVersion":      GetVersion(),
		"lintFlags":          docsFingerprintLintFlags(flags),
	}
	if enabled {
		payload["ruleset"] = docsCanonicalRuleSet(selectedRS)
		if flags != nil {
			payload["ignoreFile"] = docsPathFingerprint(flags.IgnoreFile)
			payload["functions"] = docsPathFingerprint(flags.FunctionsFlag)
		}
	}
	encoded, _ := json.Marshal(payload)
	return fmt.Sprintf("%016x", xxhash.Sum64(encoded))
}

func docsFingerprintLintFlags(flags *LintFlags) map[string]any {
	if flags == nil {
		return nil
	}
	return map[string]any{
		"hardMode":             flags.HardModeFlag,
		"turbo":                flags.TurboMode,
		"extRefs":              flags.ExtRefsFlag,
		"resolveAllRefs":       flags.ResolveAllRefs,
		"nestedRefsDocContext": flags.NestedRefsDocContext,
		"skipCheck":            flags.SkipCheckFlag,
		"remote":               flags.RemoteFlag,
		"base":                 flags.BaseFlag,
		"timeout":              flags.TimeoutFlag,
		"lookupTimeout":        flags.LookupTimeoutFlag,
		"ruleset":              flags.RulesetFlag,
		"ignoreArrayCircleRef": flags.IgnoreArrayCircleRef,
		"ignorePolyCircleRef":  flags.IgnorePolymorphCircleRef,
	}
}

type docsRuleSetFingerprint struct {
	Description      string                `json:"description,omitempty"`
	DocumentationURI string                `json:"documentationUri,omitempty"`
	Formats          []string              `json:"formats,omitempty"`
	Extends          any                   `json:"extends,omitempty"`
	Aliases          map[string]any        `json:"aliases,omitempty"`
	RuleDefinitions  map[string]any        `json:"ruleDefinitions,omitempty"`
	Rules            []docsRuleFingerprint `json:"rules,omitempty"`
}

type docsRuleFingerprint struct {
	ID               string              `json:"id,omitempty"`
	Description      string              `json:"description,omitempty"`
	DocumentationURL string              `json:"documentationUrl,omitempty"`
	Message          string              `json:"message,omitempty"`
	Given            any                 `json:"given,omitempty"`
	Formats          []string            `json:"formats,omitempty"`
	Resolved         bool                `json:"resolved,omitempty"`
	Recommended      bool                `json:"recommended,omitempty"`
	Type             string              `json:"type,omitempty"`
	Severity         string              `json:"severity,omitempty"`
	Then             any                 `json:"then,omitempty"`
	RuleCategory     *model.RuleCategory `json:"ruleCategory,omitempty"`
	HowToFix         string              `json:"howToFix,omitempty"`
	AutoFixFunction  string              `json:"autoFixFunction,omitempty"`
}

func docsCanonicalRuleSet(rs *rulesets.RuleSet) *docsRuleSetFingerprint {
	if rs == nil {
		return nil
	}
	ruleIDs := make([]string, 0, len(rs.Rules))
	for id := range rs.Rules {
		ruleIDs = append(ruleIDs, id)
	}
	sort.Strings(ruleIDs)
	rules := make([]docsRuleFingerprint, 0, len(ruleIDs))
	for _, id := range ruleIDs {
		rule := rs.Rules[id]
		if rule == nil {
			continue
		}
		rules = append(rules, docsRuleFingerprint{
			ID:               rule.Id,
			Description:      rule.Description,
			DocumentationURL: rule.DocumentationURL,
			Message:          rule.Message,
			Given:            rule.Given,
			Formats:          append([]string(nil), rule.Formats...),
			Resolved:         rule.Resolved,
			Recommended:      rule.Recommended,
			Type:             rule.Type,
			Severity:         rule.Severity,
			Then:             rule.Then,
			RuleCategory:     rule.RuleCategory,
			HowToFix:         rule.HowToFix,
			AutoFixFunction:  rule.AutoFixFunction,
		})
	}
	return &docsRuleSetFingerprint{
		Description:      rs.Description,
		DocumentationURI: rs.DocumentationURI,
		Formats:          append([]string(nil), rs.Formats...),
		Extends:          rs.Extends,
		Aliases:          cloneAnyMap(rs.Aliases),
		RuleDefinitions:  cloneAnyMap(rs.RuleDefinitions),
		Rules:            rules,
	}
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

type docsPathDigest struct {
	Path    string `json:"path,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Size    int64  `json:"size,omitempty"`
	ModTime string `json:"modTime,omitempty"`
	Error   string `json:"error,omitempty"`
}

func docsPathFingerprint(raw string) *docsPathDigest {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	resolved, err := ResolveConfigPath(raw)
	if err != nil {
		return &docsPathDigest{Path: raw, Error: err.Error()}
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return &docsPathDigest{Path: resolved, Error: err.Error()}
	}
	if info.IsDir() {
		return docsDirectoryFingerprint(resolved)
	}
	return docsFileFingerprint(resolved, info)
}

func docsFileFingerprint(path string, info os.FileInfo) *docsPathDigest {
	digest := &docsPathDigest{
		Path:    path,
		Size:    info.Size(),
		ModTime: info.ModTime().UTC().Format(time.RFC3339Nano),
	}
	data, err := os.ReadFile(path)
	if err != nil {
		digest.Error = err.Error()
		return digest
	}
	digest.Hash = fmt.Sprintf("%016x", xxhash.Sum64(data))
	return digest
}

func docsDirectoryFingerprint(root string) *docsPathDigest {
	hash := xxhash.New()
	digest := &docsPathDigest{Path: root}
	err := filepath.WalkDir(root, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(rel), len(data))
		_, _ = hash.Write(data)
		digest.Size += int64(len(data))
		return nil
	})
	if err != nil {
		digest.Error = err.Error()
		return digest
	}
	digest.Hash = fmt.Sprintf("%016x", hash.Sum64())
	return digest
}
