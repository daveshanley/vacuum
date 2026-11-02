package model

import (
	"github.com/pb33f/go-yaml"
)

// AutoFixFunction repairs a finding by mutating the supplied node or document.
// Return the supplied node or nil on success. Returning a different node is an
// error: replacement does not identify the owning parent or mapping slot.
// Errors discard all changes made by the rule transaction.
type AutoFixFunction func(node *yaml.Node, document *yaml.Node, context *RuleFunctionContext) (*yaml.Node, error)
