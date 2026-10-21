package main

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/pb33f/go-yaml"
)

// Boot registers the repair used by the CLI acceptance check.
func Boot(manager *plugin.Manager) {
	manager.RegisterAutoFixFunction("clearTitle", func(node, doc *yaml.Node, ctx *model.RuleFunctionContext) (*yaml.Node, error) {
		node.Value = ""
		return node, nil
	})
}
func main() {}
