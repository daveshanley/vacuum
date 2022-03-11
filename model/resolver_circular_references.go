package model

import (
	"fmt"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

// Reference is a wrapper around *yaml.Node results to make things more manageable when performing
// algorithms on data models. the *yaml.Node def is just a bit too low level for tracking state.
type Reference struct {
	Definition string
	Name       string
	Node       *yaml.Node
	Relations  []*Reference
	Resolved   bool
	Circular   bool
	Seen       bool
}

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey       []*Reference
	JourneyString string
	Start         *Reference
	LoopIndex     int
	LoopPoint     *Reference
}

// CheckForSchemaCircularReferences will traverse a supplied path and look cycles in the graph.
// will return any circular results (or nil) and will always return the list of node references searched, and a sequenced collection
// of the same know nodes (so repeat runs find circular references in the same order)
func CheckForSchemaCircularReferences(searchPath string, rootNode *yaml.Node) ([]*CircularReferenceResult, map[string]*Reference, []*Reference) {

	path, _ := yamlpath.NewPath(searchPath)
	results, _ := path.Find(rootNode)

	knownObjects := make(map[string]*Reference)
	var sequenceObjects []*Reference
	var name string
	if results != nil {
		for x, component := range results[0].Content {
			if x%2 == 0 {
				name = component.Value
				continue
			}

			label := strings.ReplaceAll(strings.ReplaceAll(searchPath, ".", "/"), "$", "#")
			def := fmt.Sprintf("%s/%s", label, name)
			ref := &Reference{
				Definition: def,
				Name:       name,
				Node:       component,
			}

			knownObjects[def] = ref
			sequenceObjects = append(sequenceObjects, ref)
		}
	}

	var schemasRequiringResolving []*Reference

	// remove anything that does not contain any other references, they are not required here.
	// ignore polymorphic stuff, that can create endless loops.
	for _, knownObject := range sequenceObjects {
		path, _ = yamlpath.NewPath("$.properties[*][?(@.$ref)]")
		res, _ := path.Find(knownObject.Node)

		seenRelations := make(map[string]bool)
		for _, relative := range res {
			if !seenRelations[relative.Content[1].Value] {

				// TODO: add check to make sure known object can be found.

				knownObject.Relations = append(knownObject.Relations,
					&Reference{
						Definition: relative.Content[1].Value,
						Name:       knownObjects[relative.Content[1].Value].Name,
						Node:       relative,
					},
				)
				seenRelations[relative.Content[1].Value] = true
			}
		}

		if len(res) > 0 {
			schemasRequiringResolving = append(schemasRequiringResolving, knownObject)
		}
	}

	var problems []*CircularReferenceResult
	seenProblems := make(map[string]bool)
	for _, needsResolving := range schemasRequiringResolving {
		if len(needsResolving.Relations) > 0 {

			wg := sync.WaitGroup{}
			wg.Add(len(needsResolving.Relations))

			for _, relation := range needsResolving.Relations {

				var goVisitEverything = func() {

					journey := []*Reference{needsResolving}
					problem := visitReference(relation, needsResolving, journey, knownObjects)

					if problem != nil {
						for _, p := range problem {
							if !seenProblems[p.LoopPoint.Definition] {
								seenProblems[p.LoopPoint.Definition] = true
								problems = append(problems, p)

							}
						}
					}
					knownObjects[relation.Definition].Seen = true
					wg.Done()
				}
				go goVisitEverything()
			}
			wg.Wait()
		}
	}

	return problems, knownObjects, sequenceObjects
}

func visitReference(
	reference *Reference,
	parent *Reference,
	journey []*Reference,
	knownReferences map[string]*Reference) []*CircularReferenceResult {

	locatedReference := knownReferences[reference.Definition]

	path, _ := yamlpath.NewPath("$.properties[*][?(@.$ref)]")
	res, _ := path.Find(locatedReference.Node)

	if len(res) > 0 {

		// now we need to clean these results again, there could be a ton of dupes
		filtered := make(map[string]*Reference)
		for _, rel := range res {
			dupeCheckRef := knownReferences[rel.Content[1].Value]
			if filtered[dupeCheckRef.Definition] == nil {
				filtered[dupeCheckRef.Definition] = dupeCheckRef
			}
		}
		var circularResults []*CircularReferenceResult

		for _, checkReference := range filtered {
			if !checkReference.Seen {

				pos := hasReferenceBeenSeen(checkReference, journey)

				if pos >= 0 {

					// looking at an immediate loop back?
					if checkReference.Node.Line != locatedReference.Node.Line {
						journey = append(journey, locatedReference, checkReference)
					} else {
						journey = append(journey, checkReference)
					}

					sb := strings.Builder{}
					for i, j := range journey {
						if i == pos {
							sb.WriteString(fmt.Sprintf("** %s **", j.Name))
						} else {
							sb.WriteString(fmt.Sprintf("%s", j.Name))
						}
						if i < len(journey)-1 {
							sb.WriteString(" --> ")
						}
					}

					// TODO: extract the cycle and bubble up for report.

					circRef := &CircularReferenceResult{
						Journey:       journey,
						JourneyString: sb.String(),
						Start:         journey[0],
						LoopIndex:     pos,
						LoopPoint:     journey[len(journey)-1],
					}

					checkReference.Seen = true
					checkReference.Circular = true
					return []*CircularReferenceResult{circRef}
				}

				var splitJourney = journey

				splitJourney = append(journey, locatedReference)
				circularResults = append(circularResults,
					visitReference(checkReference, parent, splitJourney, knownReferences)...)
			}
		}
		return circularResults

	}

	return nil
}

func hasReferenceBeenSeen(ref *Reference, journey []*Reference) int {
	for x, journeyRef := range journey {
		if journeyRef.Definition == ref.Definition {
			return x
		}
	}
	return -1
}
