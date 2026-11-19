// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	jsonpathConfig "github.com/pb33f/jsonpath/pkg/jsonpath/config"
	"github.com/pb33f/jsonpath/pkg/jsonpath/token"
	openapiUtils "github.com/pb33f/libopenapi/utils"
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
	contract model.Contract,
) ([]*yaml.Node, error) {
	path, err := compileRuleGivenPath(expression, cache, contract)
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
		return nil, fmt.Errorf("node lookup timeout exceeded (%v); increase --lookup-timeout to allow more time", timeout)
	}
}

func compileRuleGivenPath(expression string, cache *sync.Map, contract model.Contract) (*jsonpath.JSONPath, error) {
	cleaned := openapiUtils.FixContext(expression)
	cacheKey := struct {
		expression string
		contract   model.Contract
	}{cleaned, contract}
	if cache != nil {
		if cached, ok := cache.Load(cacheKey); ok {
			entry := cached.(cachedRuleJSONPath)
			return entry.path, entry.err
		}
	}

	selector := cleaned
	if contract == model.OpenLintV1Draft {
		selector = openLintLiteralMembers(selector)
	}
	path, err := jsonpath.NewPath(selector,
		jsonpathConfig.WithSpectralCompatibility(),
		jsonpathConfig.WithLazyContextTracking(),
	)
	if cache != nil {
		actual, _ := cache.LoadOrStore(cacheKey, cachedRuleJSONPath{path: path, err: err})
		entry := actual.(cachedRuleJSONPath)
		return entry.path, entry.err
	}
	return path, err
}

// OpenLint accepts slash-prefixed and hyphenated members after a dot. Use
// the parser's literal-member escape outside brackets and filter expressions.
func openLintLiteralMembers(expression string) string {
	if !strings.Contains(expression, "./") && !strings.Contains(expression, "-") {
		return expression
	}
	tokens := token.NewTokenizer(expression, jsonpathConfig.WithSpectralCompatibility()).Tokenize()
	var out strings.Builder
	bracketDepth, offset, copied := 0, 0, 0
	for _, t := range tokens {
		// Token lengths are in bytes and exclude whitespace between tokens.
		for offset < len(expression) && strings.ContainsRune(" \t\r\n", rune(expression[offset])) {
			offset++
		}
		switch t.Token {
		case token.BRACKET_LEFT:
			bracketDepth++
		case token.BRACKET_RIGHT:
			bracketDepth--
		case token.CHILD, token.RECURSIVE:
			start := offset + t.Len
			if bracketDepth != 0 || start >= len(expression) || expression[start] == '`' {
				break
			}
			end := start
			for end < len(expression) && !strings.ContainsRune(".[],^~\"' \t\r\n", rune(expression[end])) {
				end++
			}
			member := expression[start:end]
			if strings.HasPrefix(member, "/") || strings.Contains(member, "-") {
				out.WriteString(expression[copied:start])
				out.WriteByte('`')
				copied = start
			}
		}
		offset += t.Len
	}
	if copied == 0 {
		return expression
	}
	out.WriteString(expression[copied:])
	return out.String()
}
