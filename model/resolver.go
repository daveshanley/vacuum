// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package model

import (
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/rs/zerolog"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	localResolve int = 0
	httpResolve  int = 1
	fileResolve  int = 2
)

var seenRemoteSources = make(map[string]*yaml.Node)

// ResolveOpenAPIDocument will resolve all $ref schema nodes. Will resolve local, file based and remote nodes.
func ResolveOpenAPIDocument(rootNode *yaml.Node) (*yaml.Node, []ResolvingError) {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// before we touch anything, lets copy our root node.
	resolvedRoot := *rootNode
	var errors []ResolvingError

	// resolve components
	path, _ := yamlpath.NewPath("$..[?(@.$ref)]")
	results, _ := path.Find(&resolvedRoot)
	errors = append(errors, resolveReference(results, resolvedRoot)...)

	return &resolvedRoot, errors
}

type ResolvingError struct {
	Error error
	Node  *yaml.Node
}

func resolveReference(results []*yaml.Node, resolvedRoot yaml.Node) []ResolvingError {
	var errors []ResolvingError

	for _, result := range results {
		refKeyNode := result.Content[0]
		refValueNode := result.Content[1]

		if refKeyNode != nil && refValueNode != nil {
			if determineReferenceResolveType(refValueNode.Value) == localResolve {
				refResolved, err := lookupLocalReference(refValueNode.Value, &resolvedRoot, make(map[string]bool))
				if refResolved != nil {
					result.Content = refResolved.Content
				}
				if err != nil {
					errors = append(errors, ResolvingError{Error: err, Node: refValueNode})
				}
				continue
			}
			if determineReferenceResolveType(refValueNode.Value) == httpResolve {
				refResolved, err := lookupRemoteReference(refValueNode.Value)
				if refResolved != nil {
					result.Content = refResolved.Content
				}
				if err != nil {
					errors = append(errors, ResolvingError{Error: err, Node: refValueNode})
				}
				continue
			}
			if determineReferenceResolveType(refValueNode.Value) == fileResolve {
				refResolved, err := lookupFileReference(refValueNode.Value)
				if refResolved != nil {
					result.Content = refResolved.Content
				}
				if err != nil {
					errors = append(errors, ResolvingError{Error: err, Node: refValueNode})
				}
				continue
			}
		}
	}
	return errors
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

func lookupLocalReference(ref string, rootNode *yaml.Node, seenRefs map[string]bool) (*yaml.Node, error) {

	// create a JSONPath to look up local node.
	pathValue := fmt.Sprintf("$%s", strings.ReplaceAll(
		strings.ReplaceAll(ref, "/", "."), "#", ""))
	path, err := yamlpath.NewPath(pathValue)
	if err != nil {
		return nil, err
	}
	result, _ := path.Find(rootNode)
	if len(result) == 1 {

		// now we need to recurse over every reference.
		_, refValueNode := utils.FindFirstKeyNode("$ref", []*yaml.Node{result[0]}, 0)
		if refValueNode != nil {
			if !seenRefs[refValueNode.Value] {
				seenRefs[refValueNode.Value] = true
				return lookupLocalReference(refValueNode.Value, rootNode, seenRefs)
			} else {
				err = fmt.Errorf("'%s' contains a circular reference to '%s', "+
					"resolving will stop here", ref, refValueNode.Value)
				return result[0], err
			}
		} else {
			return result[0], nil
		}
	}
	return nil, fmt.Errorf("zero (or multiple nodes) returned for '%s'", pathValue)
}

// TODO: perform recursive lookup once resolved.
func lookupRemoteReference(ref string) (*yaml.Node, error) {

	// split string to remove file reference
	uri := strings.Split(ref, "#")

	if len(uri) != 2 {
		return nil, fmt.Errorf("unable to determine URI for remote reference: '%s'", ref)
	}
	var parsedRemoteDocument *yaml.Node
	if seenRemoteSources[uri[0]] != nil {
		parsedRemoteDocument = seenRemoteSources[uri[0]]
	} else {
		resp, err := http.Get(uri[0])
		if err != nil {
			return nil, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var remoteDoc yaml.Node
		err = yaml.Unmarshal(body, &remoteDoc)
		if err != nil {
			return nil, err
		}
		parsedRemoteDocument = &remoteDoc
		seenRemoteSources[uri[0]] = &remoteDoc
	}

	if parsedRemoteDocument == nil {
		return nil, fmt.Errorf("unable to parse remote reference: '%s'", uri[0])
	}

	// lookup item from reference by using a path query.
	query := fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))

	path, err := yamlpath.NewPath(query)
	if err != nil {
		return nil, err
	}
	result, err := path.Find(parsedRemoteDocument)
	if err != nil {
		return nil, err
	}
	if len(result) == 1 {
		return result[0], nil
	}

	return nil, nil
}

// TODO: perform recursive lookup once resolved.
func lookupFileReference(ref string) (*yaml.Node, error) {

	// split string to remove file reference
	uri := strings.Split(ref, "#")

	if len(uri) != 2 {
		return nil, fmt.Errorf("unable to determine filename for file reference: '%s'", ref)
	}

	file := strings.ReplaceAll(uri[0], "file:", "")

	var parsedRemoteDocument *yaml.Node
	if seenRemoteSources[file] != nil {
		parsedRemoteDocument = seenRemoteSources[file]
	} else {

		body, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var remoteDoc yaml.Node
		err = yaml.Unmarshal(body, &remoteDoc)
		if err != nil {
			return nil, err
		}
		parsedRemoteDocument = &remoteDoc
		seenRemoteSources[file] = &remoteDoc
	}

	if parsedRemoteDocument == nil {
		return nil, fmt.Errorf("unable to parse file reference: '%s'", file)
	}

	// lookup item from reference by using a path query.
	query := fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))

	path, err := yamlpath.NewPath(query)
	if err != nil {
		return nil, err
	}
	result, _ := path.Find(parsedRemoteDocument)
	if len(result) == 1 {
		return result[0], nil
	}

	return nil, nil
}
