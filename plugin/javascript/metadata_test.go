package javascript

import (
	"testing"
	"time"

	"github.com/pb33f/testify/require"
)

func TestMetadataTimeoutReleasesRuntime(t *testing.T) {
	function := NewJSRuleFunction("probe", `var first=true; function runRule(){return [];} function getSchema(){if(first){first=false;while(true){}}return {name:"probe"};}`).(*JSRuleFunction)
	function.SetTimeout(10 * time.Millisecond)
	require.NoError(t, function.CheckScript())
	_, err := function.GetSchemaWithError()
	require.ErrorContains(t, err, "timed out")
	schema, err := function.GetSchemaWithError()
	require.NoError(t, err)
	require.Equal(t, "probe", schema.Name)
}

func TestTopLevelScriptTimeout(t *testing.T) {
	function := NewJSRuleFunction("probe", `while(true){} function runRule(){return [];}`).(*JSRuleFunction)
	function.SetTimeout(10 * time.Millisecond)
	require.ErrorContains(t, function.CheckScript(), "timed out")
}
