package rulesets

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

func unsupportedRulesetReference(location string) error {
	normalized := strings.ReplaceAll(location, `\`, "/")
	if normalized == "spectral:arazzo" || (!strings.Contains(normalized, "://") && strings.HasSuffix(normalized, "/spectral:arazzo")) {
		return fmt.Errorf("Arazzo ruleset %q is not supported; remove spectral:arazzo when linting OpenAPI documents; Arazzo workflow linting requires an Arazzo-capable linter", location)
	}
	path := normalized
	if parsed, err := url.Parse(location); err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https") {
		path = parsed.Path
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs", ".cjs":
		return fmt.Errorf("JavaScript ruleset %q is not supported; use a YAML or JSON ruleset; for OWASP checks, replace the JavaScript ruleset with [vacuum:owasp, all] in extends; --functions <directory> loads custom functions, not JavaScript rulesets", location)
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
