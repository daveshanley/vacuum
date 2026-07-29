package languageserver

import (
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/language-server/protocol"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestConvertResultsIntoDiagnosticsIncludesExecutionErrors(t *testing.T) {
	diagnostics := ConvertResultsIntoDiagnostics(&motor.RuleSetExecutionResult{
		Errors: []error{errors.New("AsyncAPI parse failed")},
	})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, "document-error", diagnostics[0].Code.Value)
	assert.Contains(t, diagnostics[0].Message, "AsyncAPI parse failed")
}

func TestConvertResultIntoDiagnosticClampsMissingNodeColumns(t *testing.T) {
	diagnostic := ConvertResultIntoDiagnostic(&model.RuleFunctionResult{
		Message:   "invalid",
		StartNode: &yaml.Node{Line: 1, Column: 0},
		EndNode:   &yaml.Node{Line: 2, Column: 0},
	})

	assert.Equal(t, protocol.UInteger(0), diagnostic.Range.Start.Line)
	assert.Equal(t, protocol.UInteger(0), diagnostic.Range.Start.Character)
	assert.Equal(t, protocol.UInteger(1), diagnostic.Range.End.Line)
	assert.Equal(t, protocol.UInteger(0), diagnostic.Range.End.Character)
}
