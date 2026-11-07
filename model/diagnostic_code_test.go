package model

import (
	"encoding/json"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestRuleFunctionResultDiagnosticCodeNativeReports(t *testing.T) {
	for _, encode := range []struct {
		name      string
		marshal   func(any) ([]byte, error)
		unmarshal func([]byte, any) error
	}{
		{"json", json.Marshal, json.Unmarshal}, {"yaml", yaml.Marshal, yaml.Unmarshal},
	} {
		t.Run(encode.name, func(t *testing.T) {
			for _, code := range []string{"", "description-missing"} {
				data, err := encode.marshal(RuleFunctionResult{DiagnosticCode: code, Message: "A description is required", Path: "$.info"})
				require.NoError(t, err)
				var fields map[string]any
				require.NoError(t, encode.unmarshal(data, &fields))
				if code == "" {
					require.NotContains(t, fields, "diagnosticCode")
				} else {
					require.Equal(t, code, fields["diagnosticCode"])
				}
				require.Equal(t, "A description is required", fields["message"])
				require.Equal(t, "$.info", fields["path"])
			}
		})
	}
}
