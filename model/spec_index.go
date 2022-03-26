// Copyright 2022 Dave Shanley / Quobix
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

// SpecIndex is a complete pre-computed index of the entire specification. Numbers are pre-calculated and
// quick direct access to paths, operations, tags are all available. No need to walk the entire node tree in rules,
// everything is pre-walked if you need it.
type SpecIndex struct {
	allRefs                             map[string]*Reference              // all (deduplicated) refs
	rawSequencedRefs                    []*Reference                       // all raw references in sequence as they are scanned, not deduped.
	allMappedRefs                       map[string]*Reference              // these are the located mapped refs
	pathRefs                            map[string]map[string]*Reference   // all path references
	paramOpRefs                         map[string]map[string]*Reference   // params in operations.
	paramCompRefs                       map[string]*Reference              // params in components
	paramAllRefs                        map[string]*Reference              // combined components and ops
	paramInlineDuplicates               map[string][]*Reference            // inline params all with the same name
	globalTagRefs                       map[string]*Reference              // top level global tags
	securitySchemeRefs                  map[string]*Reference              // top level security schemes
	requestBodiesRefs                   map[string]*Reference              // top level request bodies
	responsesRefs                       map[string]*Reference              // top level responses
	headersRefs                         map[string]*Reference              // top level responses
	examplesRefs                        map[string]*Reference              // top level examples
	linksRefs                           map[string]map[string][]*Reference // all links
	operationTagsRefs                   map[string]map[string][]*Reference // tags found in operations
	callbackRefs                        map[string]*Reference              // top level callback refs
	externalDocumentsRef                []*Reference                       // all external documents in spec
	pathRefsLock                        sync.Mutex                         // create lock for all refs maps, we want to build data as fast as we can
	externalDocumentsCount              int                                // number of externalDocument nodes found
	operationTagsCount                  int                                // number of unique tags in operations
	globalTagsCount                     int                                // number of global tags defined
	totalTagsCount                      int                                // number unique tags in spec
	securitySchemesCount                int                                // security schemes
	globalRequestBodiesCount            int                                // component request bodies
	globalResponsesCount                int                                // component responses
	globalHeadersCount                  int                                // component headers
	globalExamplesCount                 int                                // component examples
	globalLinksCount                    int                                // component links
	globalCallbacks                     int                                // component callbacks.
	pathCount                           int                                // number of paths
	operationCount                      int                                // number of operations
	operationParamCount                 int                                // number of params defined in operations
	componentParamCount                 int                                // number of params defined in components
	componentsInlineParamUniqueCount    int                                // number of inline params with unique names
	componentsInlineParamDuplicateCount int                                // number of inline params with duplicate names
	schemaCount                         int                                // number of schemas
	refCount                            int                                // total ref count
	root                                *yaml.Node                         // the root document
	pathsNode                           *yaml.Node                         // paths node
	tagsNode                            *yaml.Node                         // tags node
	componentsNode                      *yaml.Node                         // components node
	parametersNode                      *yaml.Node                         // components/parameters node
	schemasNode                         *yaml.Node                         // components/schemas node
	securitySchemesNode                 *yaml.Node                         // components/securitySchemes node
	requestBodiesNode                   *yaml.Node                         // components/requestBodies node
	responsesNode                       *yaml.Node                         // components/responses node
	headersNode                         *yaml.Node                         // components/headers node
	examplesNode                        *yaml.Node                         // components/examples node
	linksNode                           *yaml.Node                         // components/links node
	callbacksNode                       *yaml.Node                         // components/callbacks node
	externalDocumentsNode               *yaml.Node                         // external documents node
}

var methodTypes = []string{"get", "post", "put", "patch", "options", "head", "delete"}

func runIndexFunction(funcs []func() int, wg *sync.WaitGroup) {
	for _, cFunc := range funcs {
		go func(wg *sync.WaitGroup, cf func() int) {
			cf()
			wg.Done()
		}(wg, cFunc)
	}
}

// NewSpecIndex will create a new index of an OpenAPI or Swagger spec. It's not resolved or converted into anything
// other than a raw index of every node for every content type in the specification. This process runs as fast as
// possible so dependencies looking through the tree, don't need to walk the entire thing over, and over.
func NewSpecIndex(rootNode *yaml.Node) *SpecIndex {

	index := new(SpecIndex)
	index.root = rootNode
	index.allRefs = make(map[string]*Reference)
	index.allMappedRefs = make(map[string]*Reference)
	index.pathRefs = make(map[string]map[string]*Reference)
	index.paramOpRefs = make(map[string]map[string]*Reference)
	index.operationTagsRefs = make(map[string]map[string][]*Reference)
	index.paramCompRefs = make(map[string]*Reference)
	index.paramAllRefs = make(map[string]*Reference)
	index.paramInlineDuplicates = make(map[string][]*Reference)
	index.globalTagRefs = make(map[string]*Reference)
	index.securitySchemeRefs = make(map[string]*Reference)
	index.requestBodiesRefs = make(map[string]*Reference)
	index.responsesRefs = make(map[string]*Reference)
	index.headersRefs = make(map[string]*Reference)
	index.examplesRefs = make(map[string]*Reference)
	index.linksRefs = make(map[string]map[string][]*Reference)
	index.callbackRefs = make(map[string]*Reference)

	// there is no node! return an empty index.
	if rootNode == nil {
		return index
	}

	// boot index.
	results := index.ExtractRefs(index.root)

	// pull out references
	index.ExtractComponentsFromRefs(results)
	index.ExtractExternalDocuments(index.root)
	index.GetPathCount()

	countFuncs := []func() int{
		index.GetOperationCount,
		index.GetComponentSchemaCount,
		index.GetGlobalTagsCount,
		index.GetComponentParameterCount,
		index.GetOperationsParameterCount,
	}

	var wg sync.WaitGroup
	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg) // run as fast as we can.
	wg.Wait()

	// these functions are aggregate and can only run once the rest of the model is ready
	countFuncs = []func() int{
		index.GetInlineUniqueParamCount,
		index.GetOperationTagsCount,
		index.GetGlobalLinksCount,
	}

	wg.Add(len(countFuncs))
	runIndexFunction(countFuncs, &wg) // run as fast as we can.
	wg.Wait()

	// these have final calculation dependencies
	index.GetInlineDuplicateParamCount()
	index.GetTotalTagsCount()

	return index
}

// ExtractRefs will return a deduplicated slice of references for every unique ref found in the document.
// The total number of refs, will generally be much higher, you can extract those from GetRawReferenceCount()
func (index *SpecIndex) ExtractRefs(node *yaml.Node) []*Reference {
	if node == nil {
		return nil
	}
	var found []*Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, index.ExtractRefs(n)...)
			}

			if i%2 == 0 && n.Value == "$ref" {

				// only look at scalar values, not maps (looking at you k8s)
				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				value := node.Content[i+1].Value

				segs := strings.Split(value, "/")
				name := segs[len(segs)-1]
				ref := &Reference{
					Definition: value,
					Name:       name,
					Node:       n,
				}

				// add to raw sequenced refs
				index.rawSequencedRefs = append(index.rawSequencedRefs, ref)

				// check if this is a dupe, if so, skip it, we don't care now.
				if index.allRefs[value] != nil { // seen before, skip.
					continue
				}

				if value == "" {
					fmt.Printf("why?")
				}

				index.allRefs[value] = ref
				found = append(found, ref)
			}
		}
	}
	index.refCount = len(index.allRefs)
	return found
}

// GetPathCount will return the number of paths found in the spec
func (index *SpecIndex) GetPathCount() int {
	if index.root == nil {
		return -1
	}

	if index.pathCount > 0 {
		return index.pathCount
	}
	pc := 0
	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "paths" {
				pn := index.root.Content[0].Content[i+1].Content
				index.pathsNode = index.root.Content[0].Content[i+1]
				pc = len(pn) / 2
			}
		}
	}
	index.pathCount = pc
	return pc
}

// ExtractExternalDocuments will extract the number of externalDocs nodes found in the document.
func (index *SpecIndex) ExtractExternalDocuments(node *yaml.Node) []*Reference {
	if node == nil {
		return nil
	}
	var found []*Reference
	if len(node.Content) > 0 {
		for i, n := range node.Content {
			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				found = append(found, index.ExtractExternalDocuments(n)...)
			}

			if i%2 == 0 && n.Value == "externalDocs" {
				docNode := node.Content[i+1]
				_, urlNode := utils.FindKeyNode("url", docNode.Content)
				if urlNode != nil {
					ref := &Reference{
						Definition: urlNode.Value,
						Name:       urlNode.Value,
						Node:       docNode,
					}
					index.externalDocumentsRef = append(index.externalDocumentsRef, ref)
				}
			}
		}
	}
	index.externalDocumentsCount = len(index.externalDocumentsRef)
	return found
}

// GetGlobalTagsCount will return the number of tags found in the top level 'tags' node of the document.
func (index *SpecIndex) GetGlobalTagsCount() int {
	if index.root == nil {
		return -1
	}

	if index.globalTagsCount > 0 {
		return index.globalTagsCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "tags" {
				tagsNode := index.root.Content[0].Content[i+1]
				if tagsNode != nil {
					index.tagsNode = tagsNode
					index.globalTagsCount = len(tagsNode.Content) // tags is an array, don't divide by 2.
				}
			}
		}
	}
	return index.globalTagsCount
}

// GetOperationTagsCount will return the number of operation tags found (tags referenced in operations)
func (index *SpecIndex) GetOperationTagsCount() int {
	if index.root == nil {
		return -1
	}

	if index.operationTagsCount > 0 {
		return index.operationTagsCount
	}

	// this is an aggregate count function that can only be run after operations
	// have been calculated.
	seen := make(map[string]bool)
	count := 0
	for _, path := range index.operationTagsRefs {
		for _, method := range path {
			for _, tag := range method {
				if !seen[tag.Name] {
					seen[tag.Name] = true
					count++
				}
			}
		}
	}
	index.operationTagsCount = count
	return index.operationTagsCount
}

// GetTotalTagsCount will return the number of global and operation tags found that are unique.
func (index *SpecIndex) GetTotalTagsCount() int {
	if index.root == nil {
		return -1
	}
	if index.totalTagsCount > 0 {
		return index.totalTagsCount
	}

	seen := make(map[string]bool)
	count := 0

	for _, gt := range index.globalTagRefs {
		if !seen[gt.Name] {
			seen[gt.Name] = true
			count++
		}
	}
	for _, ot := range index.operationTagsRefs {
		for _, m := range ot {
			for _, t := range m {
				if !seen[t.Name] {
					seen[t.Name] = true
					count++
				}
			}
		}
	}
	index.totalTagsCount = count
	return index.totalTagsCount
}

// GetGlobalLinksCount for each response of each operation method, multiple links can be defined
func (index *SpecIndex) GetGlobalLinksCount() int {
	if index.root == nil {
		return -1
	}

	if index.globalLinksCount > 0 {
		return index.globalLinksCount
	}

	index.pathRefsLock.Lock()
	for path, p := range index.pathRefs {
		for _, m := range p {

			// look through method for links
			links, _ := yamlpath.NewPath("$..links")
			res, _ := links.Find(m.Node)

			if len(res) > 0 {

				for _, link := range res {
					if utils.IsNodeMap(link) {

						ref := &Reference{
							Definition: m.Name,
							Name:       m.Name,
							Node:       link,
						}

						if index.linksRefs[path] == nil {
							index.linksRefs[path] = make(map[string][]*Reference)
						}
						if len(index.linksRefs[path][m.Name]) > 0 {
							index.linksRefs[path][m.Name] = append(index.linksRefs[path][m.Name], ref)
						}
						index.linksRefs[path][m.Name] = []*Reference{ref}
						index.globalLinksCount++
					}
				}
			}
		}
	}
	index.pathRefsLock.Unlock()
	return index.globalLinksCount
}

// GetRawReferenceCount will return the number of raw references located in the document.
func (index *SpecIndex) GetRawReferenceCount() int {
	return len(index.rawSequencedRefs)
}

func (index *SpecIndex) GetComponentSchemaCount() int {
	if index.root == nil {
		return -1
	}

	if index.schemaCount > 0 {
		return index.schemaCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			if n.Value == "components" {
				_, schemasNode := utils.FindKeyNode("schemas", index.root.Content[0].Content[i+1].Content)
				if schemasNode != nil {
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}
			}

			if n.Value == "definitions" {
				schemasNode := index.root.Content[0].Content[i+1]
				if schemasNode != nil {
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}
			}
		}
	}
	return index.schemaCount
}

// GetComponentParameterCount returns the number of parameter components defined
func (index *SpecIndex) GetComponentParameterCount() int {
	if index.root == nil {
		return -1
	}

	if index.componentParamCount > 0 {
		return index.componentParamCount
	}

	for i, n := range index.root.Content[0].Content {
		if i%2 == 0 {
			// openapi 3
			if n.Value == "components" {
				_, parametersNode := utils.FindKeyNode("parameters", index.root.Content[0].Content[i+1].Content)
				if parametersNode != nil {
					index.parametersNode = parametersNode
					index.componentParamCount = len(parametersNode.Content) / 2
				}
			}
			// openapi 2
			if n.Value == "parameters" {
				parametersNode := index.root.Content[0].Content[i+1]
				if parametersNode != nil {
					index.parametersNode = parametersNode
					index.componentParamCount = len(parametersNode.Content) / 2
				}
			}
		}
	}
	return index.componentParamCount
}

// GetOperationCount returns the number of operations (for all paths and) located in the document
func (index *SpecIndex) GetOperationCount() int {
	if index.root == nil {
		return -1
	}

	if index.operationCount > 0 {
		return index.operationCount
	}

	opCount := 0

	for x, p := range index.pathsNode.Content {
		if x%2 == 0 {

			method := index.pathsNode.Content[x+1]

			// extract methods for later use.
			for y, m := range method.Content {
				if y%2 == 0 {

					// check node is a valid method
					valid := false
					for _, methodType := range methodTypes {
						if m.Value == methodType {
							valid = true
						}
					}
					if valid {
						ref := &Reference{
							Definition: m.Value,
							Name:       m.Value,
							Node:       method.Content[y+1],
						}
						index.pathRefsLock.Lock()
						if index.pathRefs[p.Value] == nil {
							index.pathRefs[p.Value] = make(map[string]*Reference)
						}
						index.pathRefs[p.Value][ref.Name] = ref
						index.pathRefsLock.Unlock()
						// update
						opCount++
					}
				}
			}
		}
	}

	index.operationCount = opCount
	return opCount
}

// GetOperationsParameterCount returns the number of parameters defined in paths and operations.
// this method looks in top level (path level) and inside each operation (get, post etc.). Parameters can
// be hiding within multiple places.
func (index *SpecIndex) GetOperationsParameterCount() int {
	if index.root == nil {
		return -1
	}

	if index.operationParamCount > 0 {
		return index.operationParamCount
	}

	// parameters are sneaky, they can be in paths, in path operations or in components.
	// sometimes they are refs, sometimes they are inline definitions, just for fun.
	// some authors just LOVE to mix and match them all up.
	// check paths first
	for x, pathItemNode := range index.pathsNode.Content {
		if x%2 == 0 {

			pathPropertyNode := index.pathsNode.Content[x+1]

			// extract methods for later use.
			for y, prop := range pathPropertyNode.Content {
				if y%2 == 0 {

					// top level params
					if prop.Value == "parameters" {

						// let's look at params, check if they are refs or inline.
						params := pathPropertyNode.Content[y+1].Content
						index.scanOperationParams(params, pathItemNode)
					}

					// method level params.
					if isHttpMethod(prop.Value) {

						for z, httpMethodProp := range pathPropertyNode.Content[y+1].Content {
							if z%2 == 0 {
								if httpMethodProp.Value == "parameters" {
									params := pathPropertyNode.Content[y+1].Content[z+1].Content
									index.scanOperationParams(params, pathItemNode)
								}

								// extract operation tags if set.
								if httpMethodProp.Value == "tags" {
									tags := pathPropertyNode.Content[y+1].Content[z+1]

									if index.operationTagsRefs[pathItemNode.Value] == nil {
										index.operationTagsRefs[pathItemNode.Value] = make(map[string][]*Reference)
									}

									var tagRefs []*Reference
									for _, tagRef := range tags.Content {
										ref := &Reference{
											Definition: tagRef.Value,
											Name:       tagRef.Value,
											Node:       tagRef,
										}
										tagRefs = append(tagRefs, ref)
									}
									index.operationTagsRefs[pathItemNode.Value][prop.Value] = tagRefs
								}
							}
						}
					}
				}
			}
		}
	}

	// Now that all the paths and operations are processed, lets pick out everything from our pre
	// mapped refs and populate our ready to roll index of component params.
	for key, component := range index.allMappedRefs {
		if strings.Contains(key, "/parameters/") {
			index.paramCompRefs[key] = component
			index.paramAllRefs[key] = component
		}
	}

	// now build main index of all params by combining comp refs with inline params from operations.
	// use the namespace path:::param for inline params to identify them as inline.
	for path, params := range index.paramOpRefs {
		for pName, pValue := range params {
			if !strings.HasPrefix(pName, "#") {
				if index.paramInlineDuplicates[pName] == nil {
					index.paramInlineDuplicates[pName] = []*Reference{pValue}
				} else {
					index.paramInlineDuplicates[pName] = append(index.paramInlineDuplicates[pName], pValue)
				}
				index.paramAllRefs[fmt.Sprintf("%s:::%s", path, pName)] = pValue
			}
		}
	}
	index.operationParamCount = len(index.paramCompRefs) + len(index.paramInlineDuplicates)
	return index.operationParamCount

}

// GetInlineDuplicateParamCount returns the number of inline duplicate parameters (operation params)
func (index *SpecIndex) GetInlineDuplicateParamCount() int {
	if index.componentsInlineParamDuplicateCount > 0 {
		return index.componentsInlineParamDuplicateCount
	}
	dCount := len(index.paramInlineDuplicates) - index.countUniqueInlineDuplicates()
	index.componentsInlineParamDuplicateCount = dCount
	return dCount
}

// GetInlineUniqueParamCount returns the number of unique inline parameters (operation params)
func (index *SpecIndex) GetInlineUniqueParamCount() int {
	return index.countUniqueInlineDuplicates()
}

// ExtractComponentsFromRefs returns located components from references. The returned nodes from here
// can be used for resolving as they contain the actual object properties.
func (index *SpecIndex) ExtractComponentsFromRefs(refs []*Reference) []*Reference {
	var found []*Reference
	for _, ref := range refs {
		located := index.FindComponent(ref.Definition)
		if located != nil {
			found = append(found, located)
			index.allMappedRefs[ref.Definition] = located
		} else {
			// TODO: handle this add indexing error.
			fmt.Printf("NOT FOUND")
		}
	}
	return found
}

// FindComponent will locate a component by its reference, returns nil if nothing is found.
func (index *SpecIndex) FindComponent(componentId string) *Reference {
	if index.root == nil {
		return nil
	}

	//componentSearch := strings.ReplaceAll(strings.ReplaceAll(componentId, "/", "."), "#", "$")

	segs := strings.Split(componentId, "/")
	name := segs[len(segs)-1]

	friendlySearch := strings.ReplaceAll(fmt.Sprintf("%s['%s']", strings.Join(segs[:len(segs)-1], "."), name), "#", "$")

	path, _ := yamlpath.NewPath(friendlySearch)
	res, _ := path.Find(index.root)

	if len(res) == 1 {
		ref := &Reference{
			Definition: componentId,
			Name:       name,
			Node:       res[0],
		}

		return ref
	}
	return nil
}

/* private methods */

func (index *SpecIndex) countUniqueInlineDuplicates() int {
	if index.componentsInlineParamUniqueCount > 0 {
		return index.componentsInlineParamUniqueCount
	}
	unique := 0
	for _, p := range index.paramInlineDuplicates {
		if len(p) == 1 {
			unique++
		}
	}
	index.componentsInlineParamUniqueCount = unique
	return unique
}

func (index *SpecIndex) scanOperationParams(params []*yaml.Node, pathItemNode *yaml.Node) {
	for _, param := range params {

		// param is ref
		if len(param.Content) > 0 && param.Content[0].Value == "$ref" {

			paramRefName := param.Content[1].Value
			paramRef := index.allMappedRefs[paramRefName]

			if paramRef == nil {
				// TODO: handle mapping errors.
				continue
			}

			if index.paramOpRefs[pathItemNode.Value] == nil {
				index.paramOpRefs[pathItemNode.Value] = make(map[string]*Reference)
			}
			index.paramOpRefs[pathItemNode.Value][paramRefName] = paramRef
			continue

		} else {

			// param is inline.
			_, vn := utils.FindKeyNode("name", param.Content)
			if vn == nil {
				//TODO: handle this at somepoint.
				continue
			}

			ref := &Reference{
				Definition: vn.Value,
				Name:       vn.Value,
				Node:       param,
			}
			if index.paramOpRefs[pathItemNode.Value] == nil {
				index.paramOpRefs[pathItemNode.Value] = make(map[string]*Reference)
			}
			index.paramOpRefs[pathItemNode.Value][ref.Name] = ref
			continue
		}
	}
}

func isHttpMethod(val string) bool {
	switch strings.ToLower(val) {
	case methodTypes[0]:
		return true
	case methodTypes[1]:
		return true
	case methodTypes[2]:
		return true
	case methodTypes[3]:
		return true
	case methodTypes[4]:
		return true
	case methodTypes[5]:
		return true
	case methodTypes[6]:
		return true
	}
	return false
}
