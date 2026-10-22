package languageserver

import (
	"errors"
	"strings"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
)

// diagnosticsForRuleset handles configuration documents rejected by the API
// parser. Keep this off the successful API lint path and do not load extensions
// or execute functions while validating an editor buffer.
func diagnosticsForRuleset(content string) ([]protocol.Diagnostic, bool) {
	if !strings.Contains(content, "rules") && !strings.Contains(content, "extends") {
		return nil, false
	}
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return nil, false
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, false
	}
	root := document.Content[0]
	candidate := false
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "openapi", "swagger", "asyncapi", "arazzo":
			return nil, false
		case "rules", "extends":
			candidate = true
		}
	}
	if !candidate {
		return nil, false
	}
	diagnostics := []protocol.Diagnostic{}
	if _, err := rulesets.CreateRuleSetFromData([]byte(content)); err != nil {
		diagnostic := ConvertErrorIntoDiagnostic(err)
		diagnostic.Code = &protocol.IntegerOrString{Value: "ruleset-error"}
		diagnostics = append(diagnostics, diagnostic)
	}
	return diagnostics, true
}

// libopenapi currently exposes this rejection as an untyped error. Match every
// leaf exactly so a joined configuration or rule error cannot be discarded.
func onlyUnsupportedDocumentErrors(errs []error) bool {
	if len(errs) == 0 {
		return false
	}
	for _, err := range errs {
		if !unsupportedDocumentError(err) {
			return false
		}
	}
	return true
}

func unsupportedDocumentError(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return onlyUnsupportedDocumentErrors(joined.Unwrap())
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return unsupportedDocumentError(wrapped)
	}
	return err.Error() == "spec type not supported by libopenapi, sorry"
}
