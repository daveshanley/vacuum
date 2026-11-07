// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package runtime

import (
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func TestMarshalFixedDocumentFailsClosed(t *testing.T) {
	for _, root := range []*yaml.Node{nil, {Kind: yaml.Kind(255)}} {
		data, err := MarshalDocument(root, 1)
		require.Nil(t, data)
		require.Error(t, err)
	}
	data, err := MarshalDocument(nil, 0)
	require.NoError(t, err)
	require.Nil(t, data)
}

func TestMarshalFixedDocumentPreservesSourceFormatting(t *testing.T) {
	for _, source := range []string{
		"key:\n  enum: [a, a, b]\n",
		"key:\n    enum: [a, a, b]\n",
		"{\n    \"description\": \"<b>bold</b> & text\",\n    \"enum\": [\n        \"a\",\n        \"a\",\n        \"b\"\n    ]\n}\n",
	} {
		var root yaml.Node
		require.NoError(t, yaml.Unmarshal([]byte(source), &root))
		// A callback changes the sequence in place, retaining source node positions.
		var walk func(*yaml.Node)
		walk = func(n *yaml.Node) {
			if n.Kind == yaml.MappingNode {
				for i := 0; i < len(n.Content); i += 2 {
					if n.Content[i].Value == "enum" {
						seq := n.Content[i+1]
						seq.Content = append(seq.Content[:1], seq.Content[2:]...)
						return
					}
				}
			}
			for _, child := range n.Content {
				walk(child)
			}
		}
		walk(&root)
		actual, err := MarshalDocument(&root, 1, []byte(source))
		require.NoError(t, err)
		expected := strings.Replace(source, "[a, a, b]", "[a, b]", 1)
		expected = strings.Replace(expected, "        \"a\",\n        \"a\",", "        \"a\",", 1)
		require.Equal(t, expected, string(actual))
	}
}
