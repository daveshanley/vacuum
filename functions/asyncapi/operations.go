// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package asyncapi

import (
	"fmt"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
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

// RunRule checks that each operation names a channel in the permitted location.
func (o OperationChannel) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, source := range nodes {
		node, scope := resolveObject(source, ctx)
		if node == nil {
			continue
		}
		_, channel := mappingValue(node, "channel")
		if channel == nil {
			results = append(results, result(ctx, node, "", "Operation must define a channel."))
			continue
		}
		results = append(results, channelResults(scope, node, channel, "Operation")...)
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

// RunRule validates optional message lists against the operation's channel.
func (o OperationMessages) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, source := range nodes {
		node, scope := resolveObject(source, ctx)
		if node == nil {
			continue
		}
		_, channel := mappingValue(node, "channel")
		_, messages := mappingValue(node, "messages")
		results = append(results, channelMessageResults(scope, channel, messages, "Operation")...)
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

// RunRule validates reply fields when present, including referenced replies.
func (o OperationReply) RunRule(nodes []*yaml.Node, ctx model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for _, source := range nodes {
		node, scope := resolveObject(source, ctx)
		if node == nil {
			continue
		}
		_, reply := mappingValue(node, "reply")
		// This function also accepts reusable reply definitions selected by the ruleset.
		if strings.HasPrefix(nodePath(ctx, source, ""), "$.components.replies") {
			reply = node
		}
		if reply == nil {
			continue
		}
		reply, scope = resolveObject(reply, scope)
		if reply == nil {
			continue
		}
		if reply.Kind != yaml.MappingNode {
			results = append(results, result(ctx, reply, "", "Operation reply must be an object."))
			continue
		}
		_, channel := mappingValue(reply, "channel")
		if channel != nil {
			results = append(results, channelResults(scope, reply, channel, "Reply")...)
		}
		_, messages := mappingValue(reply, "messages")
		results = append(results, channelMessageResults(scope, channel, messages, "Reply")...)
		_, address := mappingValue(reply, "address")
		if address != nil {
			address, addressScope := resolveObject(address, scope)
			if address == nil {
				continue
			}
			_, location := mappingValue(address, "location")
			if location == nil || location.Kind != yaml.ScalarNode || location.Tag != "!!str" || location.Value == "" {
				results = append(results, result(addressScope, address, "", "Operation reply address must define a location."))
			}
			channelNode, _ := resolveObject(channel, scope)
			_, channelAddress := mappingValue(channelNode, "address")
			if channelAddress != nil && channelAddress.Tag != "!!null" {
				results = append(results, result(scope, channel, "", "Reply channel address must be null or omitted when a reply address is specified."))
			}
		}
	}
	return results
}

func channelResults(ctx model.RuleFunctionContext, owner, channel *yaml.Node, subject string) []model.RuleFunctionResult {
	ref := refValue(channel)
	base, parts, valid := referenceTokens(ref)
	rootOwned := strings.HasPrefix(nodePath(ctx, owner, ""), "$.operations")
	var message string
	switch {
	case ref == "":
		message = subject + " channel must be a Reference Object."
	case rootOwned && (!valid || referenceDocument(ctx, base) != referenceDocument(ctx, "") || len(parts) != 2 || parts[0] != "channels"):
		message = subject + " channel must reference `#/channels`."
	default:
		resolved, _ := resolveObject(channel, ctx)
		if resolved == nil {
			message = fmt.Sprintf("%s channel `%s` is not defined.", subject, ref)
		} else if resolved.Kind != yaml.MappingNode {
			message = subject + " channel must reference a Channel Object."
		}
	}
	if message == "" {
		return nil
	}
	return []model.RuleFunctionResult{result(ctx, channel, "", message)}
}

func channelMessageResults(ctx model.RuleFunctionContext, channel, messages *yaml.Node, subject string) []model.RuleFunctionResult {
	if messages == nil {
		return nil
	}
	if messages.Kind != yaml.SequenceNode {
		return []model.RuleFunctionResult{result(ctx, messages, "", subject+" messages must be an array of Reference Objects.")}
	}
	channelRef := refValue(channel)
	base, parts, valid := referenceTokens(channelRef)
	channelNode, _ := resolveObject(channel, ctx)
	_, known := mappingValue(channelNode, "messages")
	var results []model.RuleFunctionResult
	for _, item := range messages.Content {
		ref := refValue(item)
		messageBase, messageParts, messageValid := referenceTokens(ref)
		message := ""
		switch {
		case ref == "":
			message = subject + " messages must be Reference Objects."
		case channelRef == "":
			message = subject + " messages require a channel reference."
		case !valid || !messageValid || referenceDocument(ctx, base) != referenceDocument(ctx, messageBase) || len(messageParts) != len(parts)+2 || !slices.Equal(messageParts[:len(parts)], parts) || messageParts[len(parts)] != "messages":
			message = fmt.Sprintf("%s message `%s` must reference a message of channel `%s`.", subject, ref, channelRef)
		default:
			name := messageParts[len(messageParts)-1]
			_, target := mappingValue(known, name)
			if channelNode != nil && target == nil {
				message = fmt.Sprintf("%s message `%s` is not defined on channel `%s`.", subject, name, channelRef)
			}
		}
		if message != "" {
			results = append(results, result(ctx, item, "", message))
		}
	}
	return results
}

func channelMap(root *yaml.Node) *yaml.Node {
	_, channels := mappingValue(root, "channels")
	return channels
}
