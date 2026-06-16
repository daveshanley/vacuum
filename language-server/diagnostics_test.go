package languageserver

import (
	"errors"
	"testing"

	"github.com/daveshanley/vacuum/motor"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestConvertResultsIntoDiagnosticsIncludesExecutionErrors(t *testing.T) {
	diagnostics := ConvertResultsIntoDiagnostics(&motor.RuleSetExecutionResult{
		Errors: []error{errors.New("AsyncAPI parse failed")},
	})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, "document-error", diagnostics[0].Code.Value)
	assert.Contains(t, diagnostics[0].Message, "AsyncAPI parse failed")
}
