package repair

import (
	"github.com/pb33f/testify/require"
	"strings"
	"testing"
)

func TestApplyPreservesSourceFormatting(t *testing.T) {
	for _, source := range []string{
		"description: Use <b>bold</b> & stuff\nschema:\n  enum: [a, a, b]\n",
		"description: Use <b>bold</b> & stuff\nschema:\n    enum: [a, a, b]\n",
		"{\n    \"description\": \"Use <b>bold</b> & stuff\",\n    \"schema\": {\n        \"enum\": [\n            \"a\",\n            \"a\",\n            \"b\"\n        ]\n    }\n}\n",
	} {
		plan := NewPlan("source", []byte(source))
		plan.Operations = []Operation{{Op: "remove", Path: "/schema/enum/1"}}
		actual, err := Apply("source", []byte(source), plan)
		require.NoError(t, err)
		expected := strings.Replace(source, "[a, a, b]", "[a, b]", 1)
		expected = strings.Replace(expected, "            \"a\",\n            \"a\",", "            \"a\",", 1)
		require.Equal(t, expected, string(actual))
	}
}

func TestApplyPreservesIndentAfterMovingNestedMapping(t *testing.T) {
	source := []byte("title: Keep me\nschema:\n  wrapper:\n    type: string\n")
	plan := NewPlan("source", source)
	plan.Operations = []Operation{{Op: "move", From: "/schema/wrapper", Path: "/schema"}}
	actual, err := Apply("source", source, plan)
	require.NoError(t, err)
	require.Equal(t, "title: Keep me\nschema:\n  type: string\n", string(actual))
}
