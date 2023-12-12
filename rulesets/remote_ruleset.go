// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"io"
	"net/http"
	"slices"
	"strings"
)

func CheckForRemoteExtends(extends map[string]string) bool {
	for k, _ := range extends {
		if strings.HasPrefix(k, "http") {
			return true
		}
	}
	return false
}

func DownloadRemoteRuleSet(ctx context.Context, location string) (*RuleSet, error) {

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

func SniffOutAllRemoteRules(
	ctx context.Context,
	doneChan chan bool,
	rsm *ruleSetsModel,
	location string,
	visited []string,
	rs *RuleSet) {

	drs, err := DownloadRemoteRuleSet(ctx, location)

	if err != nil {
		rsm.logger.Error("cannot download remote ruleset",
			"location", location, "error", err.Error())
		return
	}

	// iterate over the remote ruleset and add the rules in
	for ruleName, ruleValue := range drs.Rules {
		rs.Rules[ruleName] = ruleValue
	}
	for ruleName, ruleValue := range drs.RuleDefinitions {
		rs.RuleDefinitions[ruleName] = ruleValue
	}

	visited = append(visited, location)

	// iterate over the extends and extract everything
	extends := drs.GetExtendsValue()

	// default and explicitly recommended
	if extends[SpectralOpenAPI] == SpectralRecommended || extends[SpectralOpenAPI] == SpectralOpenAPI {

		// suck in all recommended rules
		recommended := rsm.GenerateOpenAPIRecommendedRuleSet()
		for k, v := range recommended.Rules {
			rs.Rules[k] = v
		}
		for k, v := range recommended.RuleDefinitions {
			rs.RuleDefinitions[k] = v
		}
	}

	// all rules
	if extends[SpectralOpenAPI] == SpectralAll {
		// suck in all rules
		allRules := rsm.openAPIRuleSet
		for k, v := range allRules.Rules {
			rs.Rules[k] = v
		}
		for k, v := range allRules.RuleDefinitions {
			rs.RuleDefinitions[k] = v
		}
	}

	// no rules!
	if extends[SpectralOpenAPI] == SpectralOff {
		if rs.DocumentationURI == "" {
			rs.DocumentationURI = "https://quobix.com/vacuum/rulesets/no-rules"
		}
		rs.Rules = make(map[string]*model.Rule)
		rs.Description = fmt.Sprintf("All disabled ruleset, processing %d supplied rules", len(rs.RuleDefinitions))
	}

	// do we have remote extensions?
	if CheckForRemoteExtends(extends) {
		for k, _ := range extends {
			if strings.HasPrefix(k, "http") {

				if slices.Contains(visited, k) {
					rsm.logger.Warn("ruleset links to its self, circular rulesets are not permitted",
						"extends", k)
					return
				}

				// do down the rabbit hole.
				SniffOutAllRemoteRules(ctx, doneChan, rsm, k, visited, rs)
			}
		}
	}
	doneChan <- true
	return
}
