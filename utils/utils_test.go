package utils

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestRenderCodeSnippet(t *testing.T) {

	code := []string{"hey", "ho", "let's", "go!"}
	startNode := &yaml.Node{
		Line: 1,
	}

	rendered := RenderCodeSnippet(startNode, code, 1, 3)
	assert.Equal(t, "hey\nho\nlet's\n", rendered)

}
