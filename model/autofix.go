package model

import (
	"fmt"
	"go.yaml.in/yaml/v4"
	"log/slog"
)

// AutoFixRegistry manages auto-fix functions for rules
type AutoFixRegistry struct {
	fixes  map[string]AutoFixFunction
	logger *slog.Logger
}

// NewAutoFixRegistry creates a new auto-fix registry
func NewAutoFixRegistry(logger *slog.Logger) *AutoFixRegistry {
	return &AutoFixRegistry{
		fixes:  make(map[string]AutoFixFunction),
		logger: logger,
	}
}

// RegisterAutoFix registers an auto-fix function for a rule ID
func (r *AutoFixRegistry) RegisterAutoFix(ruleID string, fixFunc AutoFixFunction) {
	r.fixes[ruleID] = fixFunc
	if r.logger != nil {
		r.logger.Debug("Registered auto-fix function", "ruleID", ruleID)
	}
}

// GetAutoFix returns the auto-fix function for a rule ID, if it exists
func (r *AutoFixRegistry) GetAutoFix(ruleID string) (AutoFixFunction, bool) {
	fix, exists := r.fixes[ruleID]
	return fix, exists
}

// HasAutoFix checks if a rule has an auto-fix function
func (r *AutoFixRegistry) HasAutoFix(ruleID string) bool {
	_, exists := r.fixes[ruleID]
	return exists
}

// ApplyAutoFix applies an auto-fix for a rule violation
func (r *AutoFixRegistry) ApplyAutoFix(ruleID string, node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error) {
	fixFunc, exists := r.fixes[ruleID]
	if !exists {
		return node, fmt.Errorf("no auto-fix function registered for rule: %s", ruleID)
	}

	if r.logger != nil {
		r.logger.Debug("Applying auto-fix", "ruleID", ruleID)
	}

	fixedNode, err := fixFunc(node, document, context)
	if err != nil {
		if r.logger != nil {
			r.logger.Warn("Auto-fix failed", "ruleID", ruleID, "error", err)
		}
		return node, err
	}

	if r.logger != nil {
		r.logger.Debug("Auto-fix applied successfully", "ruleID", ruleID)
	}

	return fixedNode, nil
}

// GetRegisteredRules returns a list of rule IDs that have auto-fix functions
func (r *AutoFixRegistry) GetRegisteredRules() []string {
	rules := make([]string, 0, len(r.fixes))
	for ruleID := range r.fixes {
		rules = append(rules, ruleID)
	}
	return rules
}
