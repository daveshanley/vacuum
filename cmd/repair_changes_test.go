package cmd

import (
	"github.com/daveshanley/vacuum/motor"
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
	"github.com/pb33f/testify/require"
	"testing"
)

func TestRepairChangeFilterDeclinesShiftedArrayPositions(t *testing.T) {
	line := 3
	changes := &wcModel.DocumentChanges{PropertyChanges: &wcModel.PropertyChanges{Changes: []*wcModel.Change{{ChangeType: wcModel.Modified, Context: &wcModel.ChangeContext{NewLine: &line}}}}}
	result := &motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{Spec: []byte("items:\n  - a\n  - b\n  - c\n")}, ModifiedSpec: []byte("items:\n  - b\n  - c\n")}
	filter := repairChangeFilter(changes, result)
	require.False(t, filter.HasChanges(), "changed positional pointers must retain findings rather than select a different item")
}

func TestRepairChangeFilterMapsUnchangedArrayLines(t *testing.T) {
	line := 4
	changes := &wcModel.DocumentChanges{PropertyChanges: &wcModel.PropertyChanges{Changes: []*wcModel.Change{{ChangeType: wcModel.Modified, Context: &wcModel.ChangeContext{NewLine: &line}}}}}
	result := &motor.RuleSetExecutionResult{RuleSetExecution: &motor.RuleSetExecution{Spec: []byte("\nitems:\n  - a\n  - b\n")}, ModifiedSpec: []byte("items: [a, b]\n")}
	filter := repairChangeFilter(changes, result)
	require.True(t, filter.HasChanges())
	require.True(t, filter.IsLineChanged(1))
	require.False(t, filter.IsLineChanged(4))
}
