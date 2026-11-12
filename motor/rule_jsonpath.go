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
	if cache != nil {
		if cached, ok := cache.Load(cleaned); ok {
			entry := cached.(cachedRuleJSONPath)
			return entry.path, entry.err
		}
	}

	selector := cleaned
	if contract == model.OpenLintV1Draft {
		selector = openLintSlashMembers(selector)
	}
	path, err := jsonpath.NewPath(selector,
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

// OpenLint accepts a slash-prefixed member after a dot. Use the JSONPath
// parser's literal-member escape only at that token boundary.
func openLintSlashMembers(expression string) string {
	if !strings.Contains(expression, "./") {
		return expression
	}
	tokens := token.NewTokenizer(expression, jsonpathConfig.WithSpectralCompatibility()).Tokenize()
	var out strings.Builder
	bracketDepth, offset, copied := 0, 0, 0
	for i, t := range tokens {
		// Token lengths are in bytes and exclude whitespace between tokens.
		for offset < len(expression) && strings.ContainsRune(" \t\r\n", rune(expression[offset])) {
			offset++
		}
		switch t.Token {
		case token.BRACKET_LEFT:
			bracketDepth++
		case token.BRACKET_RIGHT:
			bracketDepth--
		case token.ILLEGAL:
			if bracketDepth == 0 && i > 0 && (tokens[i-1].Token == token.CHILD || tokens[i-1].Token == token.RECURSIVE) &&
				t.Literal == "/" && offset > 0 && offset < len(expression) && expression[offset-1:offset+1] == "./" {
				out.WriteString(expression[copied:offset])
				out.WriteByte('`')
				copied = offset
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
