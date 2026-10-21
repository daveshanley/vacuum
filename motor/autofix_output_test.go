package motor

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestMarshalFixedDocumentFailsClosed(t *testing.T) {
	for _, root := range []*yaml.Node{nil, {Kind: yaml.Kind(255)}} {
		data, err := marshalFixedDocument(root, 1)
		require.Nil(t, data)
		var executionErr *RuleExecutionError
		require.ErrorAs(t, err, &executionErr)
		require.Equal(t, RuleErrorAutoFix, executionErr.Kind)
	}
	data, err := marshalFixedDocument(nil, 0)
	require.NoError(t, err)
	require.Nil(t, data)
}

func TestEvidenceAloneReportsRecoveredPanic(t *testing.T) {
	e := &ExecutionEvidence{}
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		Spec:            []byte("openapi: 3.0.0\ninfo: {title: Test, version: '1'}\npaths: {}"),
		RuleSet:         &rulesets.RuleSet{Rules: map[string]*model.Rule{"panic": {Id: "panic", Given: "$", Then: model.RuleAction{Function: "test"}}}},
		CustomFunctions: map[string]model.RuleFunction{"test": &testRule{}},
	}, &ExecutionOptions{Evidence: e})
	defer result.ReleaseOwnedResources()
	require.Len(t, result.Errors, 1)
	var failure *RuleExecutionError
	require.ErrorAs(t, result.Errors[0], &failure)
	require.Equal(t, RuleErrorPanic, failure.Kind)
	require.Equal(t, "error", e.Rules[0].Status)
}

type changingSchemaProbe struct {
	configurationProbe
	schemaCalls int
}

func (p *changingSchemaProbe) GetSchemaWithError() (model.RuleFunctionSchema, error) {
	p.schemaCalls++
	if p.schemaCalls > 1 {
		return model.RuleFunctionSchema{}, fmt.Errorf("schema changed")
	}
	return p.GetSchema(), nil
}
func TestRuntimeSchemaFailureIsConfigurationError(t *testing.T) {
	p := &changingSchemaProbe{}
	_, err := p.GetSchemaWithError()
	require.NoError(t, err)
	result := ApplyRulesToRuleSetWithOptions(&RuleSetExecution{
		Spec:            []byte("openapi: 3.0.0\ninfo: {title: Test, version: '1'}\npaths: {}"),
		RuleSet:         &rulesets.RuleSet{Rules: map[string]*model.Rule{"probe": {Id: "probe", Given: "$", Then: model.RuleAction{Function: "probe"}}}},
		CustomFunctions: map[string]model.RuleFunction{"probe": p},
	}, &ExecutionOptions{ReportFunctionErrors: true})
	defer result.ReleaseOwnedResources()
	require.Len(t, result.Errors, 1)
	var failure *RuleExecutionError
	require.ErrorAs(t, result.Errors[0], &failure)
	require.Equal(t, RuleErrorConfiguration, failure.Kind)
	require.Zero(t, p.calls.Load())
}
