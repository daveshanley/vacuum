// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
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
		if isRemoteRulesetLocation(k) {
			return true
		}
	}
	return false
}

// CheckForLocalExtends checks if the extends map contains a local link
// returns true if it does, false if it does not
func CheckForLocalExtends(extends map[string]string) bool {
	for k := range extends {
		if isExternalRulesetLocation(k) {
			return true
		}
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

	if err := unsupportedRulesetReference(location); err != nil {
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
	if ruleResp.StatusCode < 200 || ruleResp.StatusCode >= 300 {
		return nil, fmt.Errorf("remote ruleset %q returned HTTP %d", location, ruleResp.StatusCode)
	}

	if ruleResp.Request != nil && ruleResp.Request.URL != nil {
		if err := unsupportedRulesetReference(ruleResp.Request.URL.String()); err != nil {
			return nil, err
		}
	}

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

	downloadedRS, rsErr := CreateRuleSetFromData(ruleBytes)
	if rsErr != nil {
		return nil, rsErr
	}

	finalURL := req.URL
	if ruleResp.Request != nil && ruleResp.Request.URL != nil {
		finalURL = ruleResp.Request.URL
	}
	if finalURL.Scheme != "http" && finalURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported remote ruleset scheme %q", finalURL.Scheme)
	}
	downloadedRS.sourceLocation = rulesetLocation{remoteURL: finalURL}
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

	if err := unsupportedRulesetReference(location); err != nil {
		return nil, err
	}

	location, err := filepath.Abs(location)
	if err != nil {
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

	downloadedRS, rsErr := CreateRuleSetFromData(ruleBytes)
	if rsErr != nil {
		return nil, rsErr
	}

	downloadedRS.sourceLocation = rulesetLocation{path: location}
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
	httpClient *http.Client) {

	target := rulesetLocation{path: location}
	if remote {
		parsed, err := url.Parse(location)
		if err != nil {
			rs.addLoadError(err)
			return
		}
		target = rulesetLocation{remoteURL: parsed}
	}
	sniffExternalRules(ctx, rsm, target, visited, rs, httpClient)
}

func sniffExternalRules(ctx context.Context, rsm *ruleSetsModel, location rulesetLocation, visited []string, rs *RuleSet, httpClient *http.Client) {

	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return
	}

	var drs *RuleSet
	var err error

	if location.remoteURL != nil {
		drs, err = DownloadRemoteRuleSet(ctx, location.remoteURL.String(), httpClient)
	} else {
		drs, err = LoadLocalRuleSet(ctx, location.path)
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		rs.addLoadError(fmt.Errorf("cannot open external ruleset %q: %w", location.String(), err))
		rsm.logger.Error("cannot open external ruleset",
			"location", location.String(), "error", err.Error())
		return
	}
	if ctx.Err() != nil {
		return
	}

	for ruleName, ruleValue := range drs.RuleDefinitions {
		if ctx.Err() != nil {
			return
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
				return
			}
			if _, exists := rs.Aliases[name]; !exists {
				rs.Aliases[name] = value
			}
		}
		rs.mutex.Unlock()
	}

	visited = append(visited, location.String())

	// iterate over the extends and extract everything
	extends := drs.GetExtendsValue()
	for _, name := range []string{VacuumArazzo, VacuumArazzoRecommended, SpectralArazzo} {
		if mode, ok := extends[name]; ok && mode != VacuumOff {
			selected := rsm.GenerateArazzoRecommendedRuleSet()
			if mode == VacuumAll {
				selected = rsm.GenerateArazzoDefaultRuleSet()
			}
			rs.mutex.Lock()
			for id, rule := range selected.Rules {
				rs.Rules[id] = rule
			}
			rs.mutex.Unlock()
		}
	}

	// default and explicitly recommended
	if (extends[SpectralOpenAPI] == VacuumRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI) ||
		(extends[VacuumOpenAPI] == VacuumRecommended || extends[VacuumOpenAPI] == VacuumOpenAPI) {

		// suck in all recommended rules
		recommended := rsm.GenerateOpenAPIRecommendedRuleSet()
		for k, v := range recommended.Rules {
			if ctx.Err() != nil {
				return
			}
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range recommended.RuleDefinitions {
			if ctx.Err() != nil {
				return
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
				return
			}
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range allRules.RuleDefinitions {
			if ctx.Err() != nil {
				return
			}
			rs.mutex.Lock()
			rs.RuleDefinitions[k] = v
			rs.mutex.Unlock()
		}
	}

	// no rules!
	if extends[SpectralOpenAPI] == VacuumOff || extends[VacuumOpenAPI] == VacuumOff {
		if ctx.Err() != nil {
			return
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
		for k := range extends {
			if ctx.Err() != nil {
				return
			}
			if isExternalRulesetLocation(k) {
				child, err := resolveRulesetLocation(drs.sourceLocation, k)
				if err != nil {
					rs.addLoadError(err)
					return
				}
				if slices.Contains(visited, child.String()) {
					rs.addLoadError(fmt.Errorf("circular ruleset extension: %s", k))
					rsm.logger.Warn("ruleset links to its self, circular rulesets are not permitted",
						"extends", k)
					return
				}

				// do down the rabbit hole.
				sniffExternalRules(ctx, rsm, child, visited, rs, httpClient)
			}
		}
	}
}

// Keep filesystem paths separate from remote URLs so a remote extension can
// never fall through to the local loader, including malformed references.
type rulesetLocation struct {
	path      string
	remoteURL *url.URL
}

func (l rulesetLocation) String() string {
	if l.remoteURL != nil {
		return l.remoteURL.String()
	}
	return l.path
}

// In-memory rulesets keep the existing working-directory behavior.
func resolveRulesetLocation(source rulesetLocation, location string) (rulesetLocation, error) {
	if source.remoteURL != nil || isRemoteRulesetLocation(location) {
		target, err := url.Parse(location)
		if err != nil {
			return rulesetLocation{}, fmt.Errorf("invalid remote ruleset reference %q: %w", location, err)
		}
		if source.remoteURL != nil {
			target = source.remoteURL.ResolveReference(target)
		}
		if target.Scheme != "http" && target.Scheme != "https" {
			return rulesetLocation{}, fmt.Errorf("unsupported remote ruleset scheme %q", target.Scheme)
		}
		return rulesetLocation{remoteURL: target}, nil
	}
	if source.path != "" && !filepath.IsAbs(location) {
		location = filepath.Join(filepath.Dir(source.path), location)
	}
	return rulesetLocation{path: filepath.Clean(location)}, nil
}

func isRemoteRulesetLocation(location string) bool {
	return strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://")
}
