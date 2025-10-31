package model

import (
	"go.yaml.in/yaml/v4"
)

// AutoFixFunction defines the signature for auto-fix functions
type AutoFixFunction func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error)
