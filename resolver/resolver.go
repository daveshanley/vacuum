// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package resolver

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
)

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey   []*model.Reference
	Start     *model.Reference
	LoopIndex int
	LoopPoint *model.Reference
}

type ResolvingError struct {
	Error error
	Node  *yaml.Node
	Path  string
}

type Resolver struct {
	specIndex          *model.SpecIndex
	resolvedRoot       yaml.Node
	circularReferences []*CircularReferenceResult
}

func NewResolver(index *model.SpecIndex) *Resolver {
	if index == nil {
		return nil
	}
	return &Resolver{
		specIndex:    index,
		resolvedRoot: *index.GetRootNode(),
	}
}

func (resolver *Resolver) ResolveComponents() []*CircularReferenceResult {
	for _, ref := range resolver.specIndex.GetMappedReferences() {
		seenReferences := make(map[string]bool)
		var journey []*model.Reference
		ref.Node.Content = resolver.VisitReference(ref, seenReferences, journey)
	}
	return resolver.circularReferences
}

func (resolver *Resolver) VisitReference(ref *model.Reference, seen map[string]bool, journey []*model.Reference) []*yaml.Node {

	if ref.Resolved {
		return ref.Node.Content
	}

	journey = append(journey, ref)
	duplicateReferences := make(map[string]*model.Reference)
	relatives := resolver.extractRelatives(ref.Node, seen, duplicateReferences)

	seen = make(map[string]bool)

	seen[ref.Definition] = true
	for _, r := range relatives {
		for _, dup := range duplicateReferences {
			if dup.Definition == r.Definition {
				continue // ignore duplicateReferences.
			}
		}

		// check if we have seen this on the journey before, if so! it's circular
		skip := false
		for i, j := range journey {
			if j.Definition == r.Definition {

				original := resolver.specIndex.GetMappedReferences()[j.Definition]
				foundDup := resolver.specIndex.GetMappedReferences()[r.Definition]
				loop := append(journey, foundDup)
				circRef := &CircularReferenceResult{
					Journey:   loop,
					Start:     original,
					LoopIndex: i,
					LoopPoint: foundDup,
				}
				ref.Circular = true // this component has a looping reference.
				resolver.circularReferences = append(resolver.circularReferences, circRef)
				skip = true

			}
		}
		if !skip {
			original := resolver.specIndex.GetMappedReferences()[r.Definition]
			resolved := resolver.VisitReference(original, seen, journey)
			r.Node.Content = resolved
		}
	}

	ref.Seen = true
	ref.Resolved = true

	return ref.Node.Content
}

func (resolver *Resolver) extractRelatives(node *yaml.Node, seenRefs map[string]bool, dups map[string]*model.Reference) []*model.Reference {

	var found []*model.Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, resolver.extractRelatives(n, seenRefs, dups)...)
			}

			if i%2 == 0 && n.Value == "$ref" {

				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				value := node.Content[i+1].Value
				ref := resolver.specIndex.GetMappedReferences()[value]
				if seenRefs[value] {
					dups[value] = ref
					continue
				} else {

					if ref == nil {
						// TODO handle error, missing ref, can't resolve.
						fmt.Println("missing ref!")
						continue
					}

					r := &model.Reference{
						Definition: value,
						Name:       value,
						Node:       node,
					}

					found = append(found, r)

					seenRefs[value] = true
				}
			}

			if i%2 == 0 && n.Value != "$ref" && n.Value != "" {

				if n.Value == "allOf" ||
					n.Value == "oneOf" ||
					n.Value == "anyOf" {

					break
				}
			}

		}
	}

	return found
}
