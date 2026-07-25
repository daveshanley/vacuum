// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// CheckForRemoteExtends checks if the extends map contains a remote link
// returns true if it does, false if it does not
func CheckForRemoteExtends(extends map[string]string) bool {
	for k := range extends {
		if strings.HasPrefix(k, "http") {
			return true
		}
	}
	return false
}

// CheckForLocalExtends checks if the extends map contains a local link
// returns true if it does, false if it does not
func CheckForLocalExtends(extends map[string]string) bool {
	for k := range extends {
		if hasRuleSetFileExtension(k) {
			return true
		}
	}
	return false
}

// ruleSetFileExtensions are the extensions recognized as a ruleset file reference. JavaScript
// module extensions are recognized so they can be reported as unsupported, rather than skipped
// silently as if the extends entry were never written.
var ruleSetFileExtensions = []string{".yml", ".yaml", ".json", ".js", ".mjs", ".cjs", ".ts"}

// hasRuleSetFileExtension reports whether a location looks like a ruleset file reference.
func hasRuleSetFileExtension(location string) bool {
	// strip any query string so remote locations such as ruleset.yaml?v=2 still match
	ext := strings.ToLower(filepath.Ext(strings.SplitN(location, "?", 2)[0]))
	return slices.Contains(ruleSetFileExtensions, ext)
}

// UnsupportedRuleSetFormatError reports a ruleset format Vacuum can identify but cannot execute.
type UnsupportedRuleSetFormatError struct {
	Location string
}

func (e UnsupportedRuleSetFormatError) Error() string {
	msg := fmt.Sprintf(`cannot load '%s': this is a JavaScript module, not a ruleset.

vacuum is not a JavaScript platform. rulesets must be YAML or JSON documents. vacuum cannot
import, bundle or execute JavaScript modules, so Spectral rulesets published as .js or .mjs
files (anything using 'export default', 'createRulesetFunction' or npm imports) will never
load, no matter how they are referenced.

JavaScript in vacuum is limited to custom functions supplied via --functions, which must be
plain scripts exposing getSchema() and runRule(). see %s/api/custom-javascript-functions
for details.`,
		e.Location, model.WebsiteUrl)

	if hint := builtInRuleSetHint(e.Location); hint != "" {
		msg = fmt.Sprintf("%s\n\n%s", msg, hint)
	}
	return msg
}

// builtInRuleSetHint suggests a native vacuum replacement when a JavaScript ruleset location
// matches one vacuum already implements, so the error ends in a fix rather than a dead end.
func builtInRuleSetHint(location string) string {
	loc := strings.ToLower(location)
	switch {
	case strings.Contains(loc, "owasp"):
		return "vacuum implements the OWASP API Security Top 10 natively, no download required.\n" +
			"replace this entry with:\n\n  extends:\n    - [spectral:owasp, all]"
	case strings.Contains(loc, "spectral-rulesets") || strings.Contains(loc, "oas-ruleset"):
		return "vacuum implements the Spectral OAS rules natively, no download required.\n" +
			"replace this entry with:\n\n  extends:\n    - [spectral:oas, all]"
	case strings.Contains(loc, "asyncapi"):
		return "vacuum implements AsyncAPI rules natively, no download required.\n" +
			"replace this entry with:\n\n  extends:\n    - [spectral:asyncapi, all]"
	}
	return "if this ruleset has no vacuum equivalent, its rules must be rewritten as YAML,\n" +
		"using vacuum's built-in functions or a custom function passed via --functions."
}

func isUnsupportedJavaScriptRuleset(location string, data []byte) bool {
	switch strings.ToLower(filepath.Ext(strings.SplitN(location, "?", 2)[0])) {
	case ".js", ".mjs", ".cjs", ".ts":
		return true
	case ".yml", ".yaml", ".json":
		// a known data format, never sniff the body. YAML rulesets are free to mention
		// JavaScript in descriptions, patterns and documentation links.
		return false
	}
	// unknown extension, the body is all we have to go on. Only a leading module
	// statement counts, so a stray mention inside a YAML string cannot trigger this.
	return startsWithJavaScriptModuleStatement(data)
}

// startsWithJavaScriptModuleStatement reports whether the first meaningful line of a
// payload is an ES module import or export, which no YAML or JSON ruleset can begin with.
func startsWithJavaScriptModuleStatement(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") ||
			strings.HasPrefix(line, "*") {
			continue
		}
		return strings.HasPrefix(line, "import ") ||
			strings.HasPrefix(line, "import{") ||
			strings.HasPrefix(line, "export ") ||
			strings.HasPrefix(line, "export{") ||
			strings.HasPrefix(line, "export default")
	}
	return false
}

// DownloadRemoteRuleSet downloads a remote ruleset and returns a *RuleSet
// returns an error if it cannot download the ruleset
func DownloadRemoteRuleSet(ctx context.Context, location string, httpClient *http.Client) (*RuleSet, error) {

	if location == "" {
		return nil, fmt.Errorf("cannot download ruleset, location is empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, "GET", location, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", location, err)
	}

	ruleResp, ruleRemoteErr := httpClient.Do(req)
	if ruleRemoteErr != nil {
		return nil, ruleRemoteErr
	}
	defer ruleResp.Body.Close()

	ruleBytes, bytesErr := io.ReadAll(ruleResp.Body)
	if bytesErr != nil {
		return nil, bytesErr
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}

	if len(ruleBytes) <= 0 {
		return nil, fmt.Errorf("remote ruleset '%s' is empty, cannot extend", location)
	}
	if isUnsupportedJavaScriptRuleset(location, ruleBytes) {
		return nil, UnsupportedRuleSetFormatError{Location: location}
	}

	downloadedRS, rsErr := CreateRuleSetFromData(ruleBytes)
	if rsErr != nil {
		return nil, rsErr
	}

	return downloadedRS, nil
}

// LoadLocalRuleSet loads a local ruleset and returns a *RuleSet
// returns an error if it cannot load the ruleset
func LoadLocalRuleSet(ctx context.Context, location string) (*RuleSet, error) {

	if location == "" {
		return nil, fmt.Errorf("cannot load ruleset, location is empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ruleBytes, bytesErr := os.ReadFile(location)
	if bytesErr != nil {
		return nil, bytesErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(ruleBytes) <= 0 {
		return nil, fmt.Errorf("local ruleset '%s' is empty, cannot extend", location)
	}
	if isUnsupportedJavaScriptRuleset(location, ruleBytes) {
		return nil, UnsupportedRuleSetFormatError{Location: location}
	}

	downloadedRS, rsErr := CreateRuleSetFromData(ruleBytes)
	if rsErr != nil {
		return nil, rsErr
	}

	return downloadedRS, nil
}

// SniffOutAllExternalRules takes a ruleset and sniffs out all external rules
// it will recursively sniff out all external rulesets and add them to the ruleset
// it will return an error if it cannot sniff out the ruleset
func SniffOutAllExternalRules(
	ctx context.Context,
	rsm *ruleSetsModel,
	location string,
	visited []string,
	rs *RuleSet,
	remote bool,
	httpClient *http.Client) []error {

	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil
	}

	var drs *RuleSet
	var err error

	if remote {
		drs, err = DownloadRemoteRuleSet(ctx, location, httpClient)
	} else {
		drs, err = LoadLocalRuleSet(ctx, location)
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		// an unsupported format already explains itself and names its location, wrapping it
		// in 'cannot open external ruleset' would only bury the explanation.
		var unsupported UnsupportedRuleSetFormatError
		if errors.As(err, &unsupported) {
			rsm.logger.Error("unsupported ruleset format", "location", location)
			return []error{err}
		}
		rsm.logger.Error("cannot open external ruleset",
			"location", location, "error", err.Error())
		return []error{fmt.Errorf("cannot open external ruleset %s: %w", location, err)}
	}
	if ctx.Err() != nil {
		return nil
	}

	for ruleName, ruleValue := range drs.RuleDefinitions {
		if ctx.Err() != nil {
			return nil
		}
		rs.mutex.Lock()
		rs.RuleDefinitions[ruleName] = mergeRuleDefinition(rs.RuleDefinitions[ruleName], ruleValue)
		rs.mutex.Unlock()
	}

	// Merge aliases from external ruleset (parent takes precedence).
	if drs.Aliases != nil {
		rs.mutex.Lock()
		if rs.Aliases == nil {
			rs.Aliases = make(map[string]interface{})
		}
		for name, value := range drs.Aliases {
			if ctx.Err() != nil {
				rs.mutex.Unlock()
				return nil
			}
			if _, exists := rs.Aliases[name]; !exists {
				rs.Aliases[name] = value
			}
		}
		rs.mutex.Unlock()
	}

	visited = append(visited, location)

	// iterate over the extends and extract everything
	extends := drs.GetExtendsValue()

	// default and explicitly recommended
	if (extends[SpectralOpenAPI] == VacuumRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI) ||
		(extends[VacuumOpenAPI] == VacuumRecommended || extends[VacuumOpenAPI] == VacuumOpenAPI) {

		// suck in all recommended rules
		recommended := rsm.GenerateOpenAPIRecommendedRuleSet()
		for k, v := range recommended.Rules {
			if ctx.Err() != nil {
				return nil
			}
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range recommended.RuleDefinitions {
			if ctx.Err() != nil {
				return nil
			}
			rs.mutex.Lock()
			rs.RuleDefinitions[k] = v
			rs.mutex.Unlock()
		}
	}

	// all rules
	if extends[SpectralOpenAPI] == VacuumAll || extends[VacuumOpenAPI] == VacuumAll {
		// suck in all rules
		allRules := rsm.openAPIRuleSet
		for k, v := range allRules.Rules {
			if ctx.Err() != nil {
				return nil
			}
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range allRules.RuleDefinitions {
			if ctx.Err() != nil {
				return nil
			}
			rs.mutex.Lock()
			rs.RuleDefinitions[k] = v
			rs.mutex.Unlock()
		}
	}

	// no rules!
	if extends[SpectralOpenAPI] == VacuumOff || extends[VacuumOpenAPI] == VacuumOff {
		if ctx.Err() != nil {
			return nil
		}
		rs.mutex.Lock()
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
		rs.mutex.Unlock()
	}

	// do we have extensions?
	if CheckForRemoteExtends(extends) || CheckForLocalExtends(extends) {
		var loadErrors []error
		for k := range extends {
			if ctx.Err() != nil {
				return loadErrors
			}
			if strings.HasPrefix(k, "http") || hasRuleSetFileExtension(k) {
				nextLocation := resolveExternalRulesetLocation(location, k)
				if slices.Contains(visited, nextLocation) {
					rsm.logger.Warn("ruleset links to its self, circular rulesets are not permitted",
						"extends", k)
					return loadErrors
				}

				// do down the rabbit hole.
				loadErrors = append(loadErrors, SniffOutAllExternalRules(ctx, rsm, nextLocation, visited, rs, strings.HasPrefix(nextLocation, "http"), httpClient)...)
			}
		}
		return loadErrors
	}
	return nil
}

func resolveExternalRulesetLocation(parentLocation, childLocation string) string {
	if childLocation == "" || strings.HasPrefix(childLocation, "http://") || strings.HasPrefix(childLocation, "https://") || filepath.IsAbs(childLocation) {
		return childLocation
	}
	if strings.HasPrefix(parentLocation, "http://") || strings.HasPrefix(parentLocation, "https://") {
		parentURL, parentErr := url.Parse(parentLocation)
		childURL, childErr := url.Parse(childLocation)
		if parentErr != nil || childErr != nil {
			return childLocation
		}
		return parentURL.ResolveReference(childURL).String()
	}
	if parentLocation == "" {
		return childLocation
	}
	return filepath.Clean(filepath.Join(filepath.Dir(parentLocation), childLocation))
}
