// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package languageserver

import (
	"net/url"
	"path/filepath"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/go-yaml"
)

// External findings remain attached to the owning document's source declarations.
// Related information points to the actual source without publishing diagnostics
// that could overwrite another open document's independently computed results.
func anchorArazzoSourceDiagnostic(result *motor.RuleSetExecutionResult, finding *model.RuleFunctionResult, diagnostic *protocol.Diagnostic) {
	if result.Arazzo == nil || finding.Origin == nil || finding.Origin.AbsoluteLocation == "" {
		return
	}
	location := finding.Origin.AbsoluteLocation
	u, err := url.Parse(location)
	if err != nil {
		return
	}
	if u.Scheme == "" {
		u = &url.URL{Scheme: "file", Path: filepath.ToSlash(location)}
	}
	diagnostic.RelatedInformation = []protocol.DiagnosticRelatedInformation{{
		Location: protocol.Location{URI: u.String(), Range: diagnostic.Range}, Message: finding.Message,
	}}
	diagnostic.Message = "In source " + location + ": " + diagnostic.Message
	root := result.Arazzo.RootNode
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	anchor := root
	if root.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "sourceDescriptions" {
				anchor = root.Content[i]
				break
			}
		}
	}
	start := protocol.Position{Line: protocol.UInteger(zeroBasedCoordinate(anchor.Line)), Character: protocol.UInteger(zeroBasedCoordinate(anchor.Column))}
	end := start
	end.Character += protocol.UInteger(len(anchor.Value))
	diagnostic.Range = protocol.Range{Start: start, End: end}
}
