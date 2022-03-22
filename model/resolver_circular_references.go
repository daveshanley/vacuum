package model

import (
	"fmt"
	"github.com/daveshanley/vacuum/utils"
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

var circLock sync.Mutex

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

		for _, result := range results {

			for x, component := range result.Content {
				if x%2 == 0 {
					name = component.Value
					continue
				}

				if utils.IsNodeMap(component) && len(component.Content) > 0 && component.Content[0].Value != "$ref" {

					if searchPath == "$.paths..items" ||
						searchPath == "$.paths..schema" ||
						searchPath == "$.paths..parameters" {
						continue // we should not be in here, we're looking at anyOf, allOf or oneOf
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

				} else {

					if determineReferenceResolveType(component.Value) == httpResolve {

						uri := strings.Split(component.Value, "#")

						if len(uri) == 2 && knownObjects[uri[1]] == nil {
							// get name from ref
							nameSegs := strings.Split(uri[1], "/")

							// extract remote node.
							node, err := lookupRemoteReference(component.Value)

							if err != nil {

								// TODO: unable to resolve file remotely.

							} else {
								ref := &Reference{
									Definition: uri[1],
									Name:       nameSegs[len(nameSegs)-1],
									Node:       node,
								}

								knownObjects[uri[1]] = ref
								sequenceObjects = append(sequenceObjects, ref)
							}
						}
					}

					if determineReferenceResolveType(component.Value) == fileResolve {

						uri := strings.Split(component.Value, "#")

						if len(uri) == 2 && knownObjects[uri[1]] == nil {
							// get name from ref
							nameSegs := strings.Split(uri[1], "/")

							// extract remote node.
							node, err := lookupFileReference(component.Value)

							if err != nil {

								// TODO: unable to resolve file locally

							} else {
								ref := &Reference{
									Definition: uri[1],
									Name:       nameSegs[len(nameSegs)-1],
									Node:       node,
								}

								knownObjects[uri[1]] = ref
								sequenceObjects = append(sequenceObjects, ref)
							}
						}
					}

				}
			}
		}
	}

	var schemasRequiringResolving []*Reference

	// remove anything that does not contain any other references, they are not required here.
	// ignore polymorphic stuff, that can create endless loops.
	for _, knownObject := range sequenceObjects {

		searchPaths := []string{
			"$.properties[*][?(@.$ref)]",
			"$..items[?(@.$ref)]",
		}

		for _, refSearchPath := range searchPaths {

			path, _ = yamlpath.NewPath(refSearchPath)
			res, _ := path.Find(knownObject.Node)

			seenRelations := make(map[string]bool)
			for _, relative := range res {
				if !seenRelations[relative.Content[1].Value] {

					// TODO: add check to make sure known object can be found.
					if knownObjects[relative.Content[1].Value] == nil {
						continue
					}

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
								circLock.Lock()
								problems = append(problems, p)
								circLock.Unlock()

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

	searchPaths := []string{
		"$..properties[*][?(@.$ref)]",
		"$..items[*][?(@.$ref)]",
	}

	var circularResults []*CircularReferenceResult

	// look through all search strings
	for _, pathString := range searchPaths {

		path, _ := yamlpath.NewPath(pathString)
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
		}
	}
	return circularResults
}

func hasReferenceBeenSeen(ref *Reference, journey []*Reference) int {
	for x, journeyRef := range journey {
		if journeyRef.Definition == ref.Definition {
			return x
		}
	}
	return -1
}

type MagicJourney struct {
	allRefs       map[string]*Reference
	allMappedRefs map[string]*Reference
	pathRefs      map[string]map[string]*Reference
	paramOpRefs   map[string]map[string]*Reference // params in operations.
	paramCompRefs map[string]*Reference            // params in

	pathCount      int
	operationCount int
	paramCount     int
	schemaCount    int
	root           *yaml.Node
	pathsNode      *yaml.Node
	componentsNode *yaml.Node
}

var methodTypes = []string{"get", "post", "put", "patch", "options", "head", "delete"}

func (mj *MagicJourney) ExtractRefs(node *yaml.Node) []*Reference {
	var found []*Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, mj.ExtractRefs(n)...)
			}

			if i%2 == 0 && n.Value == "$ref" {
				value := node.Content[i+1].Value
				if mj.allRefs[value] != nil {
					continue
				}
				segs := strings.Split(value, "/")
				name := segs[len(segs)-1]
				ref := &Reference{
					Definition: value,
					Name:       name,
					Node:       n,
				}
				mj.allRefs[value] = ref
				found = append(found, ref)
			}
		}
	}
	mj.schemaCount = len(mj.allRefs)
	return found
}

func (mj *MagicJourney) GetPathCount() int {
	if mj.root == nil {
		return -1
	}

	if mj.pathCount > 0 {
		return mj.pathCount
	}

	for i, n := range mj.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "paths" {
				pn := mj.root.Content[0].Content[i+1].Content
				mj.pathsNode = mj.root.Content[0].Content[i+1]
				pc := len(pn) / 2
				mj.pathCount = pc
				return pc
			}
		}
	}
	return 0
}

func (mj *MagicJourney) GetOperationCount() int {
	if mj.root == nil {
		return -1
	}

	if mj.operationCount > 0 {
		return mj.operationCount
	}

	opCount := 0

	for x, p := range mj.pathsNode.Content {
		if x%2 == 0 {

			method := mj.pathsNode.Content[x+1]

			// extract methods for later use.
			for y, m := range method.Content {
				if y%2 == 0 {

					// check node is a valid method
					valid := false
					for _, method := range methodTypes {
						if m.Value == method {
							valid = true
						}
					}
					if valid {
						ref := &Reference{
							Definition: m.Value,
							Name:       m.Value,
							Node:       method.Content[y+1],
						}
						if mj.pathRefs[p.Value] == nil {
							mj.pathRefs[p.Value] = make(map[string]*Reference)
							mj.pathRefs[p.Value][ref.Name] = ref
						}
						// update
						opCount++
					}
				}
			}
		}
	}

	mj.operationCount = opCount
	return opCount
}

func (mj *MagicJourney) GetParameterCount() (int, error) {
	if mj.root == nil {
		return -1, nil
	}

	if mj.paramCount > 0 {
		return mj.paramCount, nil
	}

	for x, p := range mj.pathsNode.Content {
		if x%2 == 0 {

			method := mj.pathsNode.Content[x+1]

			// extract methods for later use.
			for y, m := range method.Content {
				if y%2 == 0 {

					// top level params
					if m.Value == "parameters" {

						// let's look at params, check if they are refs or inline.

						for _, param := range method.Content[y+1].Content {

							// param is ref
							if len(param.Content) > 0 && param.Content[0].Value == "$ref" {

								paramRefName := param.Content[1].Value
								paramRef := mj.allMappedRefs[paramRefName]

								if paramRef == nil {
									// TODO: handle mapping errors.
									fmt.Printf("unknown! %s", paramRefName)
									continue
								}

								if mj.paramOpRefs[p.Value] == nil {
									mj.paramOpRefs[p.Value] = make(map[string]*Reference)
								}
								mj.paramOpRefs[p.Value][paramRefName] = paramRef
								continue

							} else {

								// param is inline.

								_, vn := utils.FindKeyNode("name", param.Content)
								if vn == nil {

									//TODO: handle error
									fmt.Printf("no name op param %s", p.Value)
									continue
								}

								ref := &Reference{
									Definition: vn.Value,
									Name:       vn.Value,
									Node:       param,
								}
								if mj.paramOpRefs[p.Value] == nil {
									mj.paramOpRefs[p.Value] = make(map[string]*Reference)
								}
								mj.paramOpRefs[p.Value][ref.Name] = ref
								continue
							}
						}
					}

					// TODO: check each method now for inline op level params.

				}
			}
		}
	}

	return 0, nil

	// extract paths

}

func (mj *MagicJourney) ExtractComponentsFromRefs(refs []*Reference) []*Reference {
	var found []*Reference
	for _, ref := range refs {
		located := mj.FindComponent(ref.Definition)
		if located != nil {
			found = append(found, located)
			mj.allMappedRefs[ref.Definition] = located
		}
	}
	return found
}

func (mj *MagicJourney) FindComponent(componentId string) *Reference {
	if mj.root == nil {
		return nil
	}

	componentSearch := strings.ReplaceAll(strings.ReplaceAll(componentId, "/", "."), "#", "$")
	path, _ := yamlpath.NewPath(componentSearch)
	res, _ := path.Find(mj.root)
	if len(res) == 1 {

		segs := strings.Split(componentSearch, "/")
		name := segs[len(segs)-1]

		ref := &Reference{
			Definition: componentId,
			Name:       name,
			Node:       res[0],
		}

		return ref
	}
	return nil
}
