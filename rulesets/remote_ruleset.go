// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"io"
	"net/http"
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
		if filepath.Ext(k) == ".yml" ||
			filepath.Ext(k) == ".yaml" ||
			filepath.Ext(k) == ".json" {
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
	httpClient *http.Client) {

	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return
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
			return
		}
		rsm.logger.Error("cannot open external ruleset",
			"location", location, "error", err.Error())
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
			if strings.HasPrefix(k, "http") ||
				filepath.Ext(k) == ".yml" ||
				filepath.Ext(k) == ".yaml" ||
				filepath.Ext(k) == ".json" {
				if slices.Contains(visited, k) {
					rsm.logger.Warn("ruleset links to its self, circular rulesets are not permitted",
						"extends", k)
					return
				}

				// do down the rabbit hole.
				SniffOutAllExternalRules(ctx, rsm, k, visited, rs, strings.HasPrefix(k, "http"), httpClient)
			}
		}
	}
}
