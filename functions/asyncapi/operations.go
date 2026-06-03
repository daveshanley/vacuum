// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"

	"github.com/daveshanley/vacuum/model"
	"go.yaml.in/yaml/v4"
)

// OperationChannel validates operation channel references.
type OperationChannel struct{}

// GetSchema returns the AsyncAPI operation-channel function schema.
func (o OperationChannel) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiOperationChannel"}
}

// GetCategory returns the AsyncAPI function category.
func (o OperationChannel) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks every operation has a channel reference to a declared channel.
func (o OperationChannel) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	_, channels := mappingValue(root, "channels")
	knownRoot := mappingKeys(channels)
	knownComponents := mappingKeys(componentMap(root, "channels"))

	var results []model.RuleFunctionResult
	for _, node := range nodes {
		key, channel := mappingValue(node, "channel")
		if channel == nil {
			results = append(results, result(context, node, nodePath(context, node, ""), "Operation must define a channel."))
			continue
		}
		ref := refValue(channel)
		rootName := rootRefName(ref, "channels")
		componentName := componentRefName(ref, "channels")
		switch {
		case rootName != "":
			if knownRoot[rootName] == nil {
				results = append(results, result(context, channel, nodePath(context, channel, ""), fmt.Sprintf("Operation channel `%s` is not defined.", ref)))
			}
		case componentName != "":
			if knownComponents[componentName] == nil {
				results = append(results, result(context, channel, nodePath(context, channel, ""), fmt.Sprintf("Operation channel `%s` is not defined.", ref)))
			}
		default:
			results = append(results, result(context, key, nodePath(context, key, ""), "Operation channel must reference `#/channels` or `#/components/channels`."))
		}
	}
	return results
}

// OperationMessages validates operation message references.
type OperationMessages struct{}

// GetSchema returns the AsyncAPI operation-messages function schema.
func (o OperationMessages) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiOperationMessages"}
}

// GetCategory returns the AsyncAPI function category.
func (o OperationMessages) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks every operation has at least one message reference and that
// component message references point to existing messages.
func (o OperationMessages) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	knownMessages := mappingKeys(componentMap(rootNode(context), "messages"))
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, messages := mappingValue(node, "messages")
		if messages == nil || messages.Kind != yaml.SequenceNode || len(messages.Content) == 0 {
			results = append(results, result(context, node, nodePath(context, node, ""), "Operation must define at least one message."))
			continue
		}
		for _, item := range messages.Content {
			ref := refValue(item)
			name := componentRefName(ref, "messages")
			if name == "" {
				results = append(results, result(context, item, nodePath(context, item, ""), "Operation messages must reference `#/components/messages`."))
				continue
			}
			if knownMessages[name] == nil {
				results = append(results, result(context, item, nodePath(context, item, ""), fmt.Sprintf("Operation message `%s` is not defined.", name)))
			}
		}
	}
	return results
}

// OperationReply validates operation reply channel, message and address fields.
type OperationReply struct{}

// GetSchema returns the AsyncAPI operation-reply function schema.
func (o OperationReply) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "asyncApiOperationReply"}
}

// GetCategory returns the AsyncAPI function category.
func (o OperationReply) GetCategory() string {
	return model.FunctionCategoryAsyncAPI
}

// RunRule checks reply objects for required channel, message and address
// semantics when a reply is present.
func (o OperationReply) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	root := rootNode(context)
	knownRootChannels := mappingKeys(channelMap(root))
	knownComponentChannels := mappingKeys(componentMap(root, "channels"))
	knownMessages := mappingKeys(componentMap(root, "messages"))

	var results []model.RuleFunctionResult
	for _, node := range nodes {
		_, reply := mappingValue(node, "reply")
		if reply == nil {
			continue
		}
		_, channel := mappingValue(reply, "channel")
		if channel == nil {
			results = append(results, result(context, reply, nodePath(context, reply, ""), "Operation reply must define a channel."))
		} else {
			ref := refValue(channel)
			rootName := rootRefName(ref, "channels")
			componentName := componentRefName(ref, "channels")
			switch {
			case rootName != "":
				if knownRootChannels[rootName] == nil {
					results = append(results, result(context, channel, nodePath(context, channel, ""), fmt.Sprintf("Reply channel `%s` is not defined.", rootName)))
				}
			case componentName != "":
				if knownComponentChannels[componentName] == nil {
					results = append(results, result(context, channel, nodePath(context, channel, ""), fmt.Sprintf("Reply channel `%s` is not defined.", componentName)))
				}
			default:
				results = append(results, result(context, channel, nodePath(context, channel, ""), "Reply channel must reference `#/channels` or `#/components/channels`."))
			}
		}

		_, messages := mappingValue(reply, "messages")
		if messages == nil || messages.Kind != yaml.SequenceNode || len(messages.Content) == 0 {
			results = append(results, result(context, reply, nodePath(context, reply, ""), "Operation reply must define at least one message."))
		} else {
			for _, item := range messages.Content {
				name := componentRefName(refValue(item), "messages")
				if name == "" {
					results = append(results, result(context, item, nodePath(context, item, ""), "Reply messages must reference `#/components/messages`."))
					continue
				}
				if knownMessages[name] == nil {
					results = append(results, result(context, item, nodePath(context, item, ""), fmt.Sprintf("Reply message `%s` is not defined.", name)))
				}
			}
		}

		_, address := mappingValue(reply, "address")
		if address != nil {
			_, location := mappingValue(address, "location")
			if location == nil || location.Value == "" {
				results = append(results, result(context, address, nodePath(context, address, ""), "Operation reply address must define a location."))
			}
		}
	}
	return results
}

func channelMap(root *yaml.Node) *yaml.Node {
	_, channels := mappingValue(root, "channels")
	return channels
}
