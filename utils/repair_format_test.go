package utils

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/testify/require"
)

func removeFormatFixtureDuplicate(t *testing.T, source []byte) *yaml.Node {
	t.Helper()
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(source, &root))
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
	return &root
}

func TestRepairYAMLRetainsIndentAndSequenceStyle(t *testing.T) {
	for _, tc := range []struct{ name, indent, seq string }{
		{"two spaces", "  ", "  "}, {"four spaces", "    ", "    "}, {"compact two", "  ", ""}, {"compact four", "    ", "  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "openapi: 3.1.0\ninfo:\n" + tc.indent + "title: Test\n" + tc.indent + "version: '1'\npaths: {}\ntags:\n" + tc.seq + "- name: pets\n" + tc.seq + "  description: Pets\ncomponents:\n" + tc.indent + "schemas:\n" + strings.Repeat(tc.indent, 2) + "Pet:\n" + strings.Repeat(tc.indent, 3) + "type: string\n" + strings.Repeat(tc.indent, 3) + "enum: [cat, cat, dog]\n"
			root := removeFormatFixtureDuplicate(t, []byte(source))
			actual, err := RenderRepairYAML(root, []byte(source), CaptureRepairYAMLFormat(root))
			require.NoError(t, err)
			require.Equal(t, strings.Replace(source, "[cat, cat, dog]", "[cat, dog]", 1), string(actual))
		})
	}
}

func TestRepairJSONRetainsIndentLiteralsNumbersAndEndings(t *testing.T) {
	raw := []byte(`{"openapi":"3.1.0","info":{"description":"Use <b>bold</b> & stuff","<key&>":"value"},"number":9007199254740993.0000000001,"enum":["cat","cat","dog"]}`)
	for _, indent := range []string{"", "  ", "    ", "\t"} {
		for _, ending := range []string{"", "\n", "\r\n"} {
			t.Run(indent+ending, func(t *testing.T) {
				source := raw
				expected := bytes.Replace(raw, []byte(`"cat","cat"`), []byte(`"cat"`), 1)
				if indent != "" {
					var s, e bytes.Buffer
					require.NoError(t, json.Indent(&s, source, "", indent))
					require.NoError(t, json.Indent(&e, expected, "", indent))
					source = s.Bytes()
					expected = e.Bytes()
				}
				if ending == "\r\n" {
					source = bytes.ReplaceAll(source, []byte{'\n'}, []byte(ending))
					expected = bytes.ReplaceAll(expected, []byte{'\n'}, []byte(ending))
				}
				source = append(source, []byte(ending)...)
				expected = append(expected, []byte(ending)...)
				actual, err := RenderRepairJSON(removeFormatFixtureDuplicate(t, source), source)
				require.NoError(t, err)
				require.Equal(t, string(expected), string(actual))
			})
		}
	}
}

func TestRepairYAMLPreservesFinalNewlines(t *testing.T) {
	for _, ending := range []string{"", "\n", "\n\n", "\r\n"} {
		source := "key:\n  enum: [a, a, b]"
		if ending == "\r\n" {
			source = strings.ReplaceAll(source, "\n", ending)
		}
		source += ending
		root := removeFormatFixtureDuplicate(t, []byte(source))
		actual, err := RenderRepairYAML(root, []byte(source), CaptureRepairYAMLFormat(root))
		require.NoError(t, err)
		require.Equal(t, strings.Replace(source, "[a, a, b]", "[a, b]", 1), string(actual))
	}
}
