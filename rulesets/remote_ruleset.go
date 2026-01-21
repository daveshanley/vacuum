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

	ruleBytes, bytesErr := io.ReadAll(ruleResp.Body)
	if bytesErr != nil {
		return nil, bytesErr
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
func LoadLocalRuleSet(_ context.Context, location string) (*RuleSet, error) {

	if location == "" {
		return nil, fmt.Errorf("cannot load ruleset, location is empty")
	}

	ruleBytes, bytesErr := os.ReadFile(location)
	if bytesErr != nil {
		return nil, bytesErr
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

	var drs *RuleSet
	var err error

	if remote {
		drs, err = DownloadRemoteRuleSet(ctx, location, httpClient)
	} else {
		drs, err = LoadLocalRuleSet(ctx, location)
	}
	if err != nil {
		rsm.logger.Error("cannot open external ruleset",
			"location", location, "error", err.Error())
		return
	}

	// iterate over the remote ruleset and add the rules in
	for ruleName, ruleValue := range drs.Rules {
		rs.mutex.Lock()
		rs.Rules[ruleName] = ruleValue
		rs.mutex.Unlock()
	}
	for ruleName, ruleValue := range drs.RuleDefinitions {
		rs.mutex.Lock()
		// Don't overwrite parent's rule definitions - they take precedence for overrides
		if _, exists := rs.RuleDefinitions[ruleName]; !exists {
			rs.RuleDefinitions[ruleName] = ruleValue
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
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range recommended.RuleDefinitions {
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
			rs.mutex.Lock()
			rs.Rules[k] = v
			rs.mutex.Unlock()
		}
		for k, v := range allRules.RuleDefinitions {
			rs.RuleDefinitions[k] = v
		}
	}

	// no rules!
	if extends[SpectralOpenAPI] == VacuumOff || extends[VacuumOpenAPI] == VacuumOff {
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
	}

	// do we have extensions?
	if CheckForRemoteExtends(extends) || CheckForLocalExtends(extends) {
		for k := range extends {
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
				SniffOutAllExternalRules(ctx, rsm, k, visited, rs, remote, httpClient)
			}
		}
	}
}
