package model

import (
	"github.com/pb33f/go-yaml"
)

// AutoFixFunction defines the signature for auto-fix functions
type AutoFixFunction func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error)
