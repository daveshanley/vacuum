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

	mapped := resolver.specIndex.GetMappedReferences()
	for _, ref := range mapped {
		seenReferences := make(map[string]bool)
		var journey []*model.Reference
		ref.Node.Content = resolver.VisitReference(ref, seenReferences, journey)
	}

	schemas := resolver.specIndex.GetAllSchemas()

	for s, schemaRef := range schemas {
		if mapped[s] == nil {
			seenReferences := make(map[string]bool)
			var journey []*model.Reference
			schemaRef.Node.Content = resolver.VisitReference(schemaRef, seenReferences, journey)
		}
	}
	return resolver.circularReferences
}

func (resolver *Resolver) VisitReference(ref *model.Reference, seen map[string]bool, journey []*model.Reference) []*yaml.Node {

	if ref.Resolved || ref.Seen {
		return ref.Node.Content
	}

	journey = append(journey, ref)
	relatives := resolver.extractRelatives(ref.Node, seen, journey)

	seen = make(map[string]bool)

	seen[ref.Definition] = true
	for _, r := range relatives {

		// check if we have seen this on the journey before, if so! it's circular
		skip := false
		for i, j := range journey {
			if j.Definition == r.Definition {

				foundDup := resolver.specIndex.GetMappedReferences()[r.Definition]

				var circRef *CircularReferenceResult
				if !foundDup.Circular {

					loop := append(journey, foundDup)
					circRef = &CircularReferenceResult{
						Journey:   loop,
						Start:     foundDup,
						LoopIndex: i,
						LoopPoint: foundDup,
					}

					foundDup.Seen = true
					foundDup.Circular = true
					resolver.circularReferences = append(resolver.circularReferences, circRef)

				}
				skip = true

			}
		}
		if !skip {
			original := resolver.specIndex.GetMappedReferences()[r.Definition]
			resolved := resolver.VisitReference(original, seen, journey)
			r.Node.Content = resolved // this is where we perform the actual resolving.
			ref.Seen = true
		}
	}
	ref.Resolved = true

	return ref.Node.Content
}

func (resolver *Resolver) extractRelatives(node *yaml.Node,
	foundRelatives map[string]bool,
	journey []*model.Reference) []*model.Reference {

	var found []*model.Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, resolver.extractRelatives(n, foundRelatives, journey)...)
			}

			if i%2 == 0 && n.Value == "$ref" {

				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				value := node.Content[i+1].Value
				ref := resolver.specIndex.GetMappedReferences()[value]

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

				foundRelatives[value] = true
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
