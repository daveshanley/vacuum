// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"io"
	"log/slog"
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
func DownloadRemoteRuleSet(_ context.Context, location string) (*RuleSet, error) {

	if location == "" {
		return nil, fmt.Errorf("cannot download ruleset, location is empty")
	}

	ruleResp, ruleRemoteErr := http.Get(location)
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

// loadLocalRuleSet loads a local ruleset but does not load it's extensions
func loadLocalRuleSet(_ context.Context, location string) (*RuleSet, error) {

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

// LoadLocalRuleSet loads a local ruleset and returns a *RuleSet
// returns an error if it cannot load the ruleset
func LoadLocalRuleSet(ctx context.Context, location string) (*RuleSet, error) {
	return SniffOutAllExternalRules(ctx, nil, location, nil, nil, false)

}

// SniffOutAllExternalRules takes a ruleset and sniffs out all external rules
// it will recursively sniff out all external rulesets and add them to the ruleset
// it will return an error if it cannot sniff out the ruleset
// If no root ruleset is provided, the ruleset from `location` is loaded and  used as the root
func SniffOutAllExternalRules(
	ctx context.Context,
	rsm *ruleSetsModel,
	location string,
	visited []string,
	rs *RuleSet,
	remote bool) (*RuleSet, error) {

	var drs *RuleSet
	var err error

	if remote {
		drs, err = DownloadRemoteRuleSet(ctx, location)
	} else {
		// On recursive calls (loading of extensions), we need to make sure the location is relative to the root ruleset
		if len(visited) > 0 && filepath.IsLocal(location) {
			rootRulesetPath := visited[0]
			rootRulestDir := filepath.Dir(rootRulesetPath)
			location = filepath.Join(rootRulestDir, location)
		}

		drs, err = loadLocalRuleSet(ctx, location)
	}
	if err != nil {
		rsm.logger.Error("cannot open external ruleset",
			"location", location, "error", err.Error())
		return rs, nil
	}

	// If not provided, assume the first loaded file is the root ruleset
	if rsm == nil {
		rsm = &ruleSetsModel{
			openAPIRuleSet: drs,
			logger:         slog.Default(),
		}
		rs = drs
	}

	// iterate over the remote ruleset and add the rules in
	for ruleName, ruleValue := range drs.Rules {
		rs.mutex.Lock()
		rs.Rules[ruleName] = ruleValue
		rs.mutex.Unlock()
	}
	for ruleName, ruleValue := range drs.RuleDefinitions {
		rs.mutex.Lock()
		rs.RuleDefinitions[ruleName] = ruleValue
		rs.mutex.Unlock()
	}

	visited = append(visited, location)

	// iterate over the extends and extract everything
	extends := drs.GetExtendsValue()

	// default and explicitly recommended
	if extends[SpectralOpenAPI] == SpectralRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI {

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
	if extends[SpectralOpenAPI] == SpectralAll {
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
	if extends[SpectralOpenAPI] == SpectralOff {
		if drs.DocumentationURI == "" {
			drs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		drs.Rules = make(map[string]*model.Rule)
		drs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
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
					return rs, nil
				}

				// go down the rabbit hole.
				_, err = SniffOutAllExternalRules(ctx, rsm, k, visited, rs, remote)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return rs, nil
}
