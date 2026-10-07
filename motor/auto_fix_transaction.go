// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package motor

import (
	"context"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
)

// autoFixState keeps callback changes private until the complete rule can publish.
// One clone per rule preserves edits shared between findings without copying the
// entire document for every finding. The gate serializes auto-fix transactions.
// Each fixing rule still copies the complete document: this cost preserves
// isolation when callbacks edit arbitrary nodes through their document argument.
type autoFixState struct {
	root      *yaml.Node
	clones    map[*yaml.Node]*yaml.Node
	originals map[*yaml.Node]*yaml.Node
	results   []model.RuleFunctionResult
	locked    bool
	failed    bool
}

// clone copies a graph once per original node, preserving cycles and aliases.
// The reverse map lets publication update original nodes without replacing their identities.
func (s *autoFixState) clone(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if cloned := s.clones[node]; cloned != nil {
		return cloned
	}
	cloned := *node
	s.clones[node] = &cloned
	s.originals[&cloned] = node
	cloned.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		cloned.Content[i] = s.clone(child)
	}
	cloned.Alias = s.clone(node.Alias)
	return &cloned
}

func applyAutoFixesToResults(ctx ruleContext, results []model.RuleFunctionResult, rfc *model.RuleFunctionContext) {
	state := ctx.autoFixState
	resolved := ctx.resolvedExecution || (ctx.rule != nil && ctx.rule.Resolved)
	callback, exists := ctx.autoFixFunctions[ctx.rule.AutoFixFunction]
	for _, result := range results {
		if result.StartNode == nil || !exists || state.failed {
			*ctx.ruleResults = append(*ctx.ruleResults, result)
			continue
		}
		target := result.StartNode
		if resolved {
			if ctx.indexUnresolved == nil {
				if !ctx.silenceLogs && ctx.logger != nil {
					ctx.logger.Warn("Auto-fix skipped: unresolved index not available", "ruleId", ctx.rule.Id)
				}
				*ctx.ruleResults = append(*ctx.ruleResults, result)
				continue
			}
			origin := ctx.indexUnresolved.FindNodeOrigin(target)
			if origin == nil || origin.Node == nil {
				if !ctx.silenceLogs && ctx.logger != nil {
					ctx.logger.Warn("Auto-fix skipped: unable to map resolved node to canonical document", "ruleId", ctx.rule.Id)
				}
				*ctx.ruleResults = append(*ctx.ruleResults, result)
				continue
			}
			target = origin.Node
		}
		if !state.locked {
			if !acquireAutoFixGate(ctx) {
				return
			}
			state.locked = true
			state.clones = make(map[*yaml.Node]*yaml.Node)
			state.originals = make(map[*yaml.Node]*yaml.Node)
			state.root = state.clone(ctx.specNodeUnresolved)
		}
		if ctx.executionContext != nil && ctx.executionContext.Err() != nil {
			return
		}
		// A finding may refer to an external source outside the root graph.
		privateTarget := state.clone(target)
		if _, err := callback(privateTarget, state.root, rfc); err != nil {
			state.failed = true
			if !ctx.silenceLogs && ctx.logger != nil {
				ctx.logger.Warn("Auto-fix failed", "ruleId", ctx.rule.Id, "error", err)
			}
			*ctx.ruleResults = append(*ctx.ruleResults, result)
			continue
		}
		state.results = append(state.results, result)
	}
}

// acquireAutoFixGate serializes private auto-fix transactions and stops waiting
// when the rule context expires. A successful acquisition must be released.
func acquireAutoFixGate(ctx ruleContext) bool {
	executionContext := ctx.executionContext
	if executionContext == nil {
		executionContext = context.Background()
	}
	if executionContext.Err() != nil {
		return false
	}
	if ctx.autoFixGate == nil {
		return true
	}
	select {
	case <-executionContext.Done():
		return false
	case <-ctx.autoFixGate:
		if executionContext.Err() != nil {
			ctx.autoFixGate <- struct{}{}
			return false
		}
		return true
	}
}

func releaseAutoFixes(ctx ruleContext) {
	if ctx.autoFixState != nil && ctx.autoFixState.locked {
		ctx.autoFixState.locked = false
		if ctx.autoFixGate != nil {
			ctx.autoFixGate <- struct{}{}
		}
	}
}

// commitAutoFixes publishes a successful rule transaction only after claiming
// the run guard. Failed callbacks retain findings without publishing their edits.
func commitAutoFixes(ctx ruleContext) {
	s := ctx.autoFixState
	if s == nil || !s.locked {
		return
	}
	if s.failed {
		*ctx.ruleResults = append(*ctx.ruleResults, s.results...)
		return
	}
	if ctx.executionContext != nil && ctx.executionContext.Err() != nil {
		return
	}
	if !ctx.runGuard.beginSharedWork() {
		return
	}

	// Translate private graph links back to caller-owned nodes. Only changed
	// nodes are written, so existing result/index pointers retain their identity.
	visited := make(map[*yaml.Node]bool)
	var originalNode func(*yaml.Node) *yaml.Node
	originalNode = func(node *yaml.Node) *yaml.Node {
		if node == nil {
			return nil
		}
		if original := s.originals[node]; original != nil {
			return original
		}
		if s.clones[node] != nil || visited[node] {
			return node
		}
		visited[node] = true
		for i, child := range node.Content {
			node.Content[i] = originalNode(child)
		}
		node.Alias = originalNode(node.Alias)
		return node
	}
	type update struct {
		target *yaml.Node
		value  yaml.Node
	}
	var updates []update
	for private, original := range s.originals {
		value := *private
		for i, child := range value.Content {
			value.Content[i] = originalNode(child)
		}
		value.Alias = originalNode(value.Alias)
		if !sameAutoFixNode(original, &value) {
			updates = append(updates, update{original, value})
		}
	}
	for _, update := range updates {
		*update.target = update.value
	}
	for _, result := range s.results {
		result.AutoFixed = true
		*ctx.fixedResults = append(*ctx.fixedResults, result)
		if !ctx.silenceLogs && ctx.logger != nil {
			ctx.logger.Debug("Auto-fix applied", "ruleId", ctx.rule.Id, "path", result.Path)
		}
	}
}

func sameAutoFixNode(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Style != b.Style || a.Tag != b.Tag || a.Value != b.Value ||
		a.Anchor != b.Anchor || a.Alias != b.Alias || a.HeadComment != b.HeadComment ||
		a.LineComment != b.LineComment || a.FootComment != b.FootComment ||
		a.Line != b.Line || a.Column != b.Column || a.Stream != b.Stream || len(a.Content) != len(b.Content) {
		return false
	}
	for i := range a.Content {
		if a.Content[i] != b.Content[i] {
			return false
		}
	}
	return true
}
