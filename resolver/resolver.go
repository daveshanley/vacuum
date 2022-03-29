// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package resolver

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"strings"
)

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey   []*model.Reference
	Start     *model.Reference
	LoopIndex int
	LoopPoint *model.Reference
}

func (c *CircularReferenceResult) GenerateJourneyPath() string {
	buf := strings.Builder{}
	for i, ref := range c.Journey {
		buf.WriteString(ref.Name)
		if i < len(c.Journey) {
			buf.WriteString(" -> ")
		}
	}
	return buf.String()
}

// ResolvingError represents an issue the resolver had trying to stitch the tree together.
type ResolvingError struct {
	Error error
	Node  *yaml.Node
	Path  string
}

// Resolver will use a *model.SpecIndex to stitch together a resolved root tree using all the discovered
// references in the doc.
type Resolver struct {
	specIndex          *model.SpecIndex
	resolvedRoot       *yaml.Node
	resolvingErrors    []*ResolvingError
	circularReferences []*CircularReferenceResult
}

// NewResolver will create a new resolver from a *model.SpecIndex
func NewResolver(index *model.SpecIndex) *Resolver {
	if index == nil {
		return nil
	}
	return &Resolver{
		specIndex:    index,
		resolvedRoot: index.GetRootNode(),
	}
}

// GetResolvingErrors returns all errors found during resolving
func (resolver *Resolver) GetResolvingErrors() []*ResolvingError {
	return resolver.resolvingErrors
}

// GetCircularErrors returns all errors found during resolving
func (resolver *Resolver) GetCircularErrors() []*CircularReferenceResult {
	return resolver.circularReferences
}

// Resolve will resolve the specification, everything that is not polymorphic and not circular, will be resolved.
// this data can get big, it results in a massive duplication of data.
func (resolver *Resolver) Resolve() []*ResolvingError {

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

	// map everything
	for _, sequenced := range resolver.specIndex.GetAllSequencedReferences() {
		locatedDef := mapped[sequenced.Definition]
		if locatedDef != nil {
			if !locatedDef.Circular && locatedDef.Seen {
				sequenced.Node.Content = locatedDef.Node.Content
			}
		}
	}

	for _, circRef := range resolver.circularReferences {
		resolver.resolvingErrors = append(resolver.resolvingErrors, &ResolvingError{
			Error: fmt.Errorf("Circular reference detected: %s", circRef.Start.Name),
			Node:  circRef.LoopPoint.Node,
			Path:  circRef.GenerateJourneyPath(),
		})
	}

	return resolver.resolvingErrors
}

// VisitReference will visit a reference as part of a journey and will return resolved nodes.
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
			r.Seen = true
			ref.Seen = true
		}
	}
	ref.Resolved = true
	ref.Seen = true

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
					_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(value)
					err := &ResolvingError{
						Error: fmt.Errorf("cannot resolve reference '%s', it's missing", value),
						Node:  nil,
						Path:  path,
					}
					resolver.resolvingErrors = append(resolver.resolvingErrors, err)
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

					// TODO: track this.

					break
				}
			}

		}
	}

	return found
}
