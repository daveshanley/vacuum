// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"sync"
	"time"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	jsonpathConfig "github.com/pb33f/jsonpath/pkg/jsonpath/config"
	openapiUtils "github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

type cachedRuleJSONPath struct {
	path *jsonpath.JSONPath
	err  error
}

type ruleJSONPathQueryResult struct {
	nodes []*yaml.Node
	err   error
}

func findRuleGivenNodes(
	node *yaml.Node,
	expression string,
	timeout time.Duration,
	cache *sync.Map,
) ([]*yaml.Node, error) {
	path, err := compileRuleGivenPath(expression, cache)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}

	result := make(chan ruleJSONPathQueryResult, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				result <- ruleJSONPathQueryResult{err: fmt.Errorf("JSONPath query panic: %v", recovered)}
			}
		}()
		result <- ruleJSONPathQueryResult{nodes: path.Query(node)}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case queryResult := <-result:
		return queryResult.nodes, queryResult.err
	case <-timer.C:
		return nil, fmt.Errorf("node lookup timeout exceeded (%v)", timeout)
	}
}

func compileRuleGivenPath(expression string, cache *sync.Map) (*jsonpath.JSONPath, error) {
	cleaned := openapiUtils.FixContext(expression)
	if cache != nil {
		if cached, ok := cache.Load(cleaned); ok {
			entry := cached.(cachedRuleJSONPath)
			return entry.path, entry.err
		}
	}

	path, err := jsonpath.NewPath(cleaned,
		jsonpathConfig.WithSpectralCompatibility(),
		jsonpathConfig.WithLazyContextTracking(),
	)
	if cache != nil {
		actual, _ := cache.LoadOrStore(cleaned, cachedRuleJSONPath{path: path, err: err})
		entry := actual.(cachedRuleJSONPath)
		return entry.path, entry.err
	}
	return path, err
}
