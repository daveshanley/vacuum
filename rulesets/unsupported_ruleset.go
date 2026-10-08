package rulesets

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func unsupportedRulesetReference(location string) error {
	normalized := strings.ReplaceAll(location, `\`, "/")
	path := normalized
	if parsed, err := url.Parse(location); err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		path = parsed.Path
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs", ".cjs":
		return fmt.Errorf("JavaScript ruleset %q is not supported; use YAML or JSON; see https://github.com/daveshanley/vacuum#spectral-migration", location)
	}
	return nil
}

func validateRulesetReferences(rs *RuleSet) error {
	for location := range rs.GetExtendsValue() {
		if err := unsupportedRulesetReference(location); err != nil {
			return err
		}
	}
	return nil
}
