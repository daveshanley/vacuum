// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"strings"
	"sync"
)

const (
	localResolve int = 0
	httpResolve  int = 1
	fileResolve  int = 2
)

var seenRemoteSources = make(map[string]*yaml.Node)

// ResolveOpenAPIDocument will resolve all $ref schema nodes. Will resolve local, file based and remote nodes.
func ResolveOpenAPIDocument(rootNode *yaml.Node) (*yaml.Node, []ResolvingError) {

	// before we touch anything, lets copy our root node.
	resolvedRoot := *rootNode

	searchPaths := []string{
		"$.components.schemas",
		"$.components.parameters",
		"$.components.responses",
		"$.components.requestBodies",
		"$.components.examples",
		"$.components.headers",
		"$.components.securitySchemes",
		"$.components.links",
		"$.components.callbacks",
		"$.definitions",
		"$.parameters",
		"$.paths..schema",
		"$.paths..items",
		"$.paths..parameters",
	}

	knownSchemas := make(map[string]*Reference)
	var errors []ResolvingError

	var m sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(searchPaths))

	for _, path := range searchPaths {

		// check in separate threads.
		go func(path string, wg *sync.WaitGroup, known map[string]*Reference, errors *[]ResolvingError) {

			schemas, errs := checkForCircularReferences(rootNode, path)

			m.Lock()
			*errors = append(*errors, errs...)
			m.Unlock()

			for k, schema := range schemas {
				m.Lock()
				known[k] = schema
				m.Unlock()
			}
			wg.Done()
		}(path, &wg, knownSchemas, &errors)
	}

	wg.Wait()

	// now we know all the things, we can resolve everything that needs resolving!
	// performing in place resolving and searching causes all kinds of issues.
	searchPaths = []string{
		"$.paths..schema",
		"$.paths..items",
		"$.paths..parameters",
	}

	wg.Add(len(searchPaths))
	for _, path := range searchPaths {

		// check in separate threads.
		go func(path string, wg *sync.WaitGroup, known map[string]*Reference, errors *[]ResolvingError) {
			errs := resolveOperations(path, &resolvedRoot, known)

			m.Lock()
			*errors = append(*errors, errs...)
			m.Unlock()
			wg.Done()

		}(path, &wg, knownSchemas, &errors)
	}

	wg.Wait()

	return &resolvedRoot, errors
}

func resolveOperations(searchPath string, resolvedRoot *yaml.Node,
	knownSchemas map[string]*Reference) []ResolvingError {
	path, _ := yamlpath.NewPath(searchPath)
	results, _ := path.Find(resolvedRoot)

	var errors []ResolvingError
	for _, schema := range results {

		if utils.IsNodeArray(schema) {
			for _, arrayNode := range schema.Content {
				for k, n := range arrayNode.Content {
					if k%2 == 0 && n.Value == "$ref" {
						name := arrayNode.Content[k+1].Value
						errors = append(errors, resolve(name, knownSchemas, schema, errors)...)
					}
				}
			}
		} else {
			if schema.Content[0].Value == "$ref" {
				name := schema.Content[1].Value
				errors = append(errors, resolve(name, knownSchemas, schema, errors)...)
			}
		}

	}
	return errors
}

func resolve(name string, knownSchemas map[string]*Reference, schema *yaml.Node, errors []ResolvingError) []ResolvingError {
	var key string
	if determineReferenceResolveType(name) == localResolve {
		key = name
	}

	if determineReferenceResolveType(name) == httpResolve || determineReferenceResolveType(name) == fileResolve {
		keys := strings.Split(name, "#")
		if len(keys) == 2 {
			key = keys[1]
		}
	}

	resolvedSchema := knownSchemas[key]
	if resolvedSchema != nil {
		schema.Content = resolvedSchema.Node.Content
	} else {
		errors = append(errors, ResolvingError{
			Error: fmt.Errorf("component '%s' cannot be resolved", key),
			Node:  schema,
			Path:  key, // TODO: come back and make sure this is correct.
		})
	}
	return errors
}

func checkForCircularReferences(rootNode *yaml.Node, searchPath string) (map[string]*Reference, []ResolvingError) {

	circRefs, knownComponents, sequencedComponents := CheckForSchemaCircularReferences(searchPath, rootNode)

	var errors []ResolvingError

	// add circular reference errors
	if len(circRefs) > 0 {
		for _, circRef := range circRefs {
			errors = append(errors, ResolvingError{
				Error: fmt.Errorf("Circular reference detected: %s", circRef.JourneyString),
				Node:  circRef.LoopPoint.Node,
				Path:  searchPath,
			})
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(len(sequencedComponents))
	for _, comp := range sequencedComponents {

		// resolve every component in a new thread for speed.
		go func(comp *Reference) {
			resolveComponent(comp, knownComponents)
			wg.Done()
		}(comp)
	}
	wg.Wait()
	return knownComponents, errors
}

type ResolvingError struct {
	Error error
	Node  *yaml.Node
	Path  string
}

func resolveComponent(reference *Reference, known map[string]*Reference) {

	// if this is a circular component, stop resolving, the schema cannot be rendered any further.
	if reference.Circular {
		return
	}
	if len(reference.Relations) > 0 {
		for _, relation := range reference.Relations {

			// if this is a known relation
			knownRelation := known[relation.Definition]
			if knownRelation != nil {

				relation.Node.Content = knownRelation.Node.Content
				// continue resolving.
				resolveComponent(knownRelation, known)
			} else {

				// TODO: handle this
				fmt.Print("unknown, needs further processing") // check type and if no dice, we need an error.

			}
		}
		reference.Resolved = true
	}

}

func determineReferenceResolveType(ref string) int {
	if ref != "" && ref[0] == '#' {
		return localResolve
	}
	if ref != "" && len(ref) >= 5 && (ref[:5] == "https" || ref[:5] == "http:") {
		return httpResolve
	}
	if strings.Contains(ref, ".json") ||
		strings.Contains(ref, ".yaml") ||
		strings.Contains(ref, ".yml") {
		return fileResolve
	}
	return -1
}

//func lookupLocalReference(ref string, rootNode *yaml.Node, seenRefs map[string]bool) (*yaml.Node, error) {
//
//	// create a JSONPath to look up local node.
//	pathValue := fmt.Sprintf("$%s", strings.ReplaceAll(
//		strings.ReplaceAll(ref, "/", "."), "#", ""))
//	path, err := yamlpath.NewPath(pathValue)
//	if err != nil {
//		return nil, err
//	}
//	result, _ := path.Find(rootNode)
//	if len(result) == 1 {
//
//		// now we need to recurse over every reference.
//		_, refValueNode := utils.FindFirstKeyNode("$ref", []*yaml.Node{result[0]}, 0)
//		if refValueNode != nil {
//			if !seenRefs[refValueNode.Value] {
//				seenRefs[refValueNode.Value] = true
//				return lookupLocalReference(refValueNode.Value, rootNode, seenRefs)
//			} else {
//				err = fmt.Errorf("'%s' contains a circular reference to '%s', "+
//					"resolving will stop here", ref, refValueNode.Value)
//				return result[0], err
//			}
//		} else {
//			return result[0], nil
//		}
//	}
//	return nil, fmt.Errorf("zero (or multiple nodes) returned for '%s'", pathValue)
//}

var remoteLock sync.Mutex
