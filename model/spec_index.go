// Copyright 2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package model

import (
	"errors"
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

const (
	localResolve int = 0
	httpResolve  int = 1
	fileResolve  int = 2
)

// Reference is a wrapper around *yaml.Node results to make things more manageable when performing
// algorithms on data models. the *yaml.Node def is just a bit too low level for tracking state.
type Reference struct {
	Definition string
	Name       string
	Node       *yaml.Node
	Resolved   bool
	Circular   bool
	Seen       bool
	Path       string // this won't always be available.
}

// CircularReferenceResult contains a circular reference found when traversing the graph.
type CircularReferenceResult struct {
	Journey       []*Reference
	JourneyString string
	Start         *Reference
	LoopIndex     int
	LoopPoint     *Reference
}

// ReferenceMapped is a helper struct for mapped references put into sequence (we lose the key)
type ReferenceMapped struct {
	Reference  *Reference
	Definition string
}

// SpecIndex is a complete pre-computed index of the entire specification. Numbers are pre-calculated and
// quick direct access to paths, operations, tags are all available. No need to walk the entire node tree in rules,
// everything is pre-walked if you need it.
type SpecIndex struct {
	allRefs                             map[string]*Reference                       // all (deduplicated) refs
	rawSequencedRefs                    []*Reference                                // all raw references in sequence as they are scanned, not deduped.
	allMappedRefs                       map[string]*Reference                       // these are the located mapped refs
	allMappedRefsSequenced              []*ReferenceMapped                          // sequenced mapped refs
	pathRefs                            map[string]map[string]*Reference            // all path references
	paramOpRefs                         map[string]map[string]map[string]*Reference // params in operations.
	paramCompRefs                       map[string]*Reference                       // params in components
	paramAllRefs                        map[string]*Reference                       // combined components and ops
	paramInlineDuplicates               map[string][]*Reference                     // inline params all with the same name
	globalTagRefs                       map[string]*Reference                       // top level global tags
	securitySchemeRefs                  map[string]*Reference                       // top level security schemes
	requestBodiesRefs                   map[string]*Reference                       // top level request bodies
	responsesRefs                       map[string]*Reference                       // top level responses
	headersRefs                         map[string]*Reference                       // top level responses
	examplesRefs                        map[string]*Reference                       // top level examples
	linksRefs                           map[string]map[string][]*Reference          // all links
	operationTagsRefs                   map[string]map[string][]*Reference          // tags found in operations
	operationDescriptionRefs            map[string]map[string]*Reference            // descriptions in operations.
	operationSummaryRefs                map[string]map[string]*Reference            // summaries in operations
	callbackRefs                        map[string]*Reference                       // top level callback refs
	polymorphicRefs                     map[string]*Reference                       // every reference to 'allOf' references
	externalDocumentsRef                []*Reference                                // all external documents in spec
	refsWithSiblings                    map[string]*Reference                       // references with sibling elements next to them.
	pathRefsLock                        sync.Mutex                                  // create lock for all refs maps, we want to build data as fast as we can
	externalDocumentsCount              int                                         // number of externalDocument nodes found
	operationTagsCount                  int                                         // number of unique tags in operations
	globalTagsCount                     int                                         // number of global tags defined
	totalTagsCount                      int                                         // number unique tags in spec
	securitySchemesCount                int                                         // security schemes
	globalRequestBodiesCount            int                                         // component request bodies
	globalResponsesCount                int                                         // component responses
	globalHeadersCount                  int                                         // component headers
	globalExamplesCount                 int                                         // component examples
	globalLinksCount                    int                                         // component links
	globalCallbacks                     int                                         // component callbacks.
	pathCount                           int                                         // number of paths
	operationCount                      int                                         // number of operations
	operationParamCount                 int                                         // number of params defined in operations
	componentParamCount                 int                                         // number of params defined in components
	componentsInlineParamUniqueCount    int                                         // number of inline params with unique names
	componentsInlineParamDuplicateCount int                                         // number of inline params with duplicate names
	schemaCount                         int                                         // number of schemas
	refCount                            int                                         // total ref count
	root                                *yaml.Node                                  // the root document
	pathsNode                           *yaml.Node                                  // paths node
	tagsNode                            *yaml.Node                                  // tags node
	componentsNode                      *yaml.Node                                  // components node
	parametersNode                      *yaml.Node                                  // components/parameters node
	allParametersNode                   map[string]*Reference                       // all parameters node
	allParameters                       map[string]*Reference                       // all parameters (components/defs)
	schemasNode                         *yaml.Node                                  // components/schemas node
	allSchemas                          map[string]*Reference                       // all schemas
	securitySchemesNode                 *yaml.Node                                  // components/securitySchemes node
	allSecuritySchemes                  map[string]*Reference                       // all security schemes / definitions.
	requestBodiesNode                   *yaml.Node                                  // components/requestBodies node
	allRequestBodies                    map[string]*Reference                       // all request bodies
	responsesNode                       *yaml.Node                                  // components/responses node
	allResponses                        map[string]*Reference                       // all responses
	headersNode                         *yaml.Node                                  // components/headers node
	allHeaders                          map[string]*Reference                       // all headers
	examplesNode                        *yaml.Node                                  // components/examples node
	allExamples                         map[string]*Reference                       // all components examples
	linksNode                           *yaml.Node                                  // components/links node
	allLinks                            map[string]*Reference                       // all links
	callbacksNode                       *yaml.Node                                  // components/callbacks node
	allCallbacks                        map[string]*Reference                       // all components examples
	externalDocumentsNode               *yaml.Node                                  // external documents node
	allExternalDocuments                map[string]*Reference                       // all external documents
	externalSpecIndex                   map[string]*SpecIndex                       // create a primary index of all external specs and componentIds
	refErrors                           []*IndexingError                            // errors when indexing references
	operationParamErrors                []*IndexingError                            // errors when indexing parameters
	allDescriptions                     []*DescriptionReference                     // every single description found in the spec.
	allSummaries                        []*DescriptionReference                     // every single summary found in the spec.
	allEnums                            []*EnumReference                            // every single enum found in the spec.
	enumCount                           int
	descriptionCount                    int
	summaryCount                        int
	seenRemoteSources                   map[string]*yaml.Node
	remoteLock                          sync.Mutex
}

// ExternalLookupFunction is for lookup functions that take a JSONSchema reference and tries to find that node in the
// URI based document. Decides if the reference is local, remote or in a file.
type ExternalLookupFunction func(id string) (foundNode *yaml.Node, rootNode *yaml.Node, lookupError error)

// IndexingError holds data about something that went wrong during indexing.
type IndexingError struct {
	Error error
	Node  *yaml.Node
	Path  string
}

// DescriptionReference holds data about a description that was found and where it was found.
type DescriptionReference struct {
	Content   string
	Path      string
	Node      *yaml.Node
	IsSummary bool
}

type EnumReference struct {
	Node *yaml.Node
	Type *yaml.Node
	Path string
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
	index.paramOpRefs = make(map[string]map[string]map[string]*Reference)
	index.operationTagsRefs = make(map[string]map[string][]*Reference)
	index.operationDescriptionRefs = make(map[string]map[string]*Reference)
	index.operationSummaryRefs = make(map[string]map[string]*Reference)
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
	index.externalSpecIndex = make(map[string]*SpecIndex)
	index.allSchemas = make(map[string]*Reference)
	index.allParameters = make(map[string]*Reference)
	index.allSecuritySchemes = make(map[string]*Reference)
	index.allRequestBodies = make(map[string]*Reference)
	index.allResponses = make(map[string]*Reference)
	index.allHeaders = make(map[string]*Reference)
	index.allExamples = make(map[string]*Reference)
	index.allLinks = make(map[string]*Reference)
	index.allCallbacks = make(map[string]*Reference)
	index.allExternalDocuments = make(map[string]*Reference)
	index.polymorphicRefs = make(map[string]*Reference)
	index.refsWithSiblings = make(map[string]*Reference)
	index.seenRemoteSources = make(map[string]*yaml.Node)

	// there is no node! return an empty index.
	if rootNode == nil {
		return index
	}

	// boot index.
	results := index.ExtractRefs(index.root.Content[0], []string{}, 0, false)

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
	index.GetAllDescriptionsCount()
	index.GetTotalTagsCount()

	return index
}

// GetRootNode returns document root node.
func (index *SpecIndex) GetRootNode() *yaml.Node {
	if index == nil {
		return nil
	}
	return index.root
}

// GetGlobalTagsNode returns document root node.
func (index *SpecIndex) GetGlobalTagsNode() *yaml.Node {
	if index == nil {
		return nil
	}
	return index.tagsNode
}

// GetPathsNode returns document root node.
func (index *SpecIndex) GetPathsNode() *yaml.Node {
	if index == nil {
		return nil
	}
	return index.pathsNode
}

// GetDiscoveredReferences will return all unique references found in the spec
func (index *SpecIndex) GetDiscoveredReferences() map[string]*Reference {
	return index.allRefs
}

// GetPolyReferences will return every polymorphic reference in the doc
func (index *SpecIndex) GetPolyReferences() map[string]*Reference {
	return index.polymorphicRefs
}

// GetAllCombinedReferences will return the number of unique and polymorphic references discovered.
func (index *SpecIndex) GetAllCombinedReferences() map[string]*Reference {
	combined := make(map[string]*Reference)
	for k, ref := range index.allRefs {
		combined[k] = ref
	}
	for k, ref := range index.polymorphicRefs {
		combined[k] = ref
	}
	return combined
}

// GetMappedReferences will return all references that were mapped successfully to actual property nodes.
// this collection is completely unsorted, traversing it may produce random results when resolving it and
// encountering circular references can change results depending on where in the collection the resolver started
// its journey through the index.
func (index *SpecIndex) GetMappedReferences() map[string]*Reference {
	return index.allMappedRefs
}

// GetMappedReferencesSequenced will return all references that were mapped successfully to nodes, performed in sequence
// as they were read in from the document.
func (index *SpecIndex) GetMappedReferencesSequenced() []*ReferenceMapped {
	return index.allMappedRefsSequenced
}

// GetOperationParameterReferences will return all references to operation parameters
func (index *SpecIndex) GetOperationParameterReferences() map[string]map[string]map[string]*Reference {
	return index.paramOpRefs
}

// GetAllSchemas will return all schemas found in the document
func (index *SpecIndex) GetAllSchemas() map[string]*Reference {
	return index.allSchemas
}

// GetAllSecuritySchemes will return all security schemes / definitions found in the document.
func (index *SpecIndex) GetAllSecuritySchemes() map[string]*Reference {
	return index.allSecuritySchemes
}

// GetAllHeaders will return all headers found in the document (under components)
func (index *SpecIndex) GetAllHeaders() map[string]*Reference {
	return index.allHeaders
}

// GetAllExamples will return all examples found in the document (under components)
func (index *SpecIndex) GetAllExamples() map[string]*Reference {
	return index.allExamples
}

// GetAllDescriptions will return all descriptions found in the document
func (index *SpecIndex) GetAllDescriptions() []*DescriptionReference {
	return index.allDescriptions
}

// GetAllEnums will return all enums found in the document
func (index *SpecIndex) GetAllEnums() []*EnumReference {
	return index.allEnums
}

// GetAllSummaries will return all summaries found in the document
func (index *SpecIndex) GetAllSummaries() []*DescriptionReference {
	return index.allSummaries
}

// GetAllRequestBodies will return all requestBodies found in the document (under components)
func (index *SpecIndex) GetAllRequestBodies() map[string]*Reference {
	return index.allRequestBodies
}

// GetAllLinks will return all links found in the document (under components)
func (index *SpecIndex) GetAllLinks() map[string]*Reference {
	return index.allLinks
}

// GetAllParameters will return all parameters found in the document (under components)
func (index *SpecIndex) GetAllParameters() map[string]*Reference {
	return index.allParameters
}

// GetAllResponses will return all responses found in the document (under components)
func (index *SpecIndex) GetAllResponses() map[string]*Reference {
	return index.allResponses
}

// GetAllCallbacks will return all links found in the document (under components)
func (index *SpecIndex) GetAllCallbacks() map[string]*Reference {
	return index.allCallbacks
}

// GetInlineOperationDuplicateParameters will return a map of duplicates located in operation parameters.
func (index *SpecIndex) GetInlineOperationDuplicateParameters() map[string][]*Reference {
	return index.paramInlineDuplicates
}

// GetReferencesWithSiblings will return a map of all the references with sibling nodes (illegal)
func (index *SpecIndex) GetReferencesWithSiblings() map[string]*Reference {
	return index.refsWithSiblings
}

// GetAllReferences will return every reference found in the spec, after being de-duplicated.
func (index *SpecIndex) GetAllReferences() map[string]*Reference {
	return index.allRefs
}

// GetAllSequencedReferences will return every reference (in sequence) that was found (non-polymorphic)
func (index *SpecIndex) GetAllSequencedReferences() []*Reference {
	return index.rawSequencedRefs
}

// GetSchemasNode will return the schema's node found in the spec
func (index *SpecIndex) GetSchemasNode() *yaml.Node {
	return index.schemasNode
}

// GetParametersNode will return the schema's node found in the spec
func (index *SpecIndex) GetParametersNode() *yaml.Node {
	return index.parametersNode
}

// GetOperationParametersIndexErrors any errors that occurred when indexing operation parameters
func (index *SpecIndex) GetOperationParametersIndexErrors() []*IndexingError {
	return index.operationParamErrors
}

func (index *SpecIndex) checkPolymorphicNode(name string) (bool, string) {
	switch name {
	case "anyOf":
		return true, "anyOf"
	case "allOf":
		return true, "allOf"
	case "oneOf":
		return true, "oneOf"
	}
	return false, ""
}

// ExtractRefs will return a deduplicated slice of references for every unique ref found in the document.
// The total number of refs, will generally be much higher, you can extract those from GetRawReferenceCount()
func (index *SpecIndex) ExtractRefs(node *yaml.Node, seenPath []string, level int, poly bool) []*Reference {
	if node == nil {
		return nil
	}
	var found []*Reference
	if len(node.Content) > 0 {
		var prev string
		for i, n := range node.Content {

			if utils.IsNodeMap(n) || utils.IsNodeArray(n) {
				level++
				// check if we're using  polymorphic values. These tend to create rabbit warrens of circular
				// references if every single link is followed. We don't resolve polymorphic values.
				isPoly, _ := index.checkPolymorphicNode(prev)
				if isPoly {
					poly = true
				}
				found = append(found, index.ExtractRefs(n, seenPath, level, poly)...)
			}

			if i%2 == 0 && n.Value == "$ref" {

				// only look at scalar values, not maps (looking at you k8s)
				if !utils.IsNodeStringValue(node.Content[i+1]) {
					continue
				}

				fp := make([]string, len(seenPath))
				for x, foundPathNode := range seenPath {
					fp[x] = foundPathNode
				}

				value := node.Content[i+1].Value

				segs := strings.Split(value, "/")
				name := segs[len(segs)-1]
				ref := &Reference{
					Definition: value,
					Name:       name,
					Node:       node,
					Path:       fmt.Sprintf("$.%s", strings.Join(seenPath, ".")),
				}

				// add to raw sequenced refs
				index.rawSequencedRefs = append(index.rawSequencedRefs, ref)

				// if this ref value has any siblings (node.Content is larger than two elements)
				// then add to refs with siblings
				if len(node.Content) > 2 {
					index.refsWithSiblings[value] = ref
				}

				// if this is a polymorphic reference, we're going to leave it out
				// allRefs (we don't ever want these resolved, so instead of polluting
				// the timeline, we will keep each poly ref in its own collection for later
				// analysis.
				if poly {
					index.polymorphicRefs[value] = ref
					continue
				}

				// check if this is a dupe, if so, skip it, we don't care now.
				if index.allRefs[value] != nil { // seen before, skip.
					continue
				}

				if value == "" {

					completedPath := fmt.Sprintf("$.%s", strings.Join(fp, "."))

					indexError := &IndexingError{
						Error: errors.New("schema reference is empty and cannot be processed"),
						Node:  node.Content[i+1],
						Path:  completedPath,
					}

					index.refErrors = append(index.refErrors, indexError)

					continue
				}

				index.allRefs[value] = ref
				found = append(found, ref)
			}

			if i%2 == 0 && n.Value != "$ref" && n.Value != "" {

				nodePath := fmt.Sprintf("$.%s", strings.Join(seenPath, "."))

				// capture descriptions and summaries
				if n.Value == "description" {
					ref := &DescriptionReference{
						Content:   node.Content[i+1].Value,
						Path:      nodePath,
						Node:      node.Content[i+1],
						IsSummary: false,
					}

					index.allDescriptions = append(index.allDescriptions, ref)
					index.descriptionCount++
				}

				if n.Value == "summary" {
					ref := &DescriptionReference{
						Content:   node.Content[i+1].Value,
						Path:      nodePath,
						Node:      node.Content[i+1],
						IsSummary: true,
					}

					index.allSummaries = append(index.allSummaries, ref)
					index.summaryCount++
				}

				// capture enums
				if n.Value == "enum" {

					// all enums need to have a type, extract the type from the node where the enum was found.
					_, enumKeyValueNode := utils.FindKeyNode("type", node.Content)

					if enumKeyValueNode != nil {
						ref := &EnumReference{
							Path: nodePath,
							Node: node.Content[i+1],
							Type: enumKeyValueNode,
						}

						index.allEnums = append(index.allEnums, ref)
						index.enumCount++
					}
				}

				seenPath = append(seenPath, n.Value)
				prev = n.Value
			}

			// if next node is map, don't add segment.
			if i < len(node.Content)-1 {
				next := node.Content[i+1]

				if i%2 != 0 && next != nil && !utils.IsNodeArray(next) && !utils.IsNodeMap(next) {
					seenPath = seenPath[:len(seenPath)-1]
				}
			}
		}
		if len(seenPath) > 0 {
			seenPath = seenPath[:len(seenPath)-1]
		}

	}
	if len(seenPath) > 0 {
		seenPath = seenPath[:len(seenPath)-1]
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
					for x, tagNode := range index.tagsNode.Content {

						_, name := utils.FindKeyNode("name", tagNode.Content)
						_, description := utils.FindKeyNode("description", tagNode.Content)

						var desc string
						if description == nil {
							desc = ""
						}
						if name != nil {
							ref := &Reference{
								Definition: desc,
								Name:       name.Value,
								Node:       tagNode,
								Path:       fmt.Sprintf("$.tags[%d]", x),
							}
							index.globalTagRefs[name.Value] = ref
						}
					}
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
		// TODO: do we still need this?
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

				for _, link := range res[0].Content {
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

// GetComponentSchemaCount will return the number of schemas located in the 'components' or 'definitions' node.
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

				// while we are here, go ahead and extract everything in components.
				_, parametersNode := utils.FindKeyNode("parameters", index.root.Content[0].Content[i+1].Content)
				_, requestBodiesNode := utils.FindKeyNode("requestBodies", index.root.Content[0].Content[i+1].Content)
				_, responsesNode := utils.FindKeyNode("responses", index.root.Content[0].Content[i+1].Content)
				_, securitySchemesNode := utils.FindKeyNode("securitySchemes", index.root.Content[0].Content[i+1].Content)
				_, headersNode := utils.FindKeyNode("headers", index.root.Content[0].Content[i+1].Content)
				_, examplesNode := utils.FindKeyNode("examples", index.root.Content[0].Content[i+1].Content)
				_, linksNode := utils.FindKeyNode("links", index.root.Content[0].Content[i+1].Content)
				_, callbacksNode := utils.FindKeyNode("callbacks", index.root.Content[0].Content[i+1].Content)

				// extract schemas
				if schemasNode != nil {
					index.extractDefinitionsAndSchemas(schemasNode, "#/components/schemas/")
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}

				// extract parameters
				if parametersNode != nil {
					index.extractComponentParameters(parametersNode, "#/components/parameters/")
					index.parametersNode = parametersNode
				}

				// extract requestBodies
				if requestBodiesNode != nil {
					index.extractComponentRequestBodies(requestBodiesNode, "#/components/requestBodies/")
					index.requestBodiesNode = requestBodiesNode
				}

				// extract responses
				if responsesNode != nil {
					index.extractComponentResponses(responsesNode, "#/components/responses/")
					index.responsesNode = responsesNode
				}

				// extract requestBodies
				if securitySchemesNode != nil {
					index.extractComponentSecuritySchemes(securitySchemesNode, "#/components/securitySchemes/")
					index.securitySchemesNode = securitySchemesNode
				}

				// extract headers
				if headersNode != nil {
					index.extractComponentHeaders(headersNode, "#/components/headers/")
					index.headersNode = headersNode
				}

				// extract examples
				if examplesNode != nil {
					index.extractComponentExamples(examplesNode, "#/components/examples/")
					index.examplesNode = examplesNode
				}

				// extract links
				if linksNode != nil {
					index.extractComponentLinks(linksNode, "#/components/links/")
					index.linksNode = linksNode
				}

				// extract callbacks
				if callbacksNode != nil {
					index.extractComponentCallbacks(callbacksNode, "#/components/callbacks/")
					index.callbacksNode = callbacksNode
				}

			}

			if n.Value == "definitions" {
				schemasNode := index.root.Content[0].Content[i+1]
				if schemasNode != nil {

					// extract schemas
					index.extractDefinitionsAndSchemas(schemasNode, "#/definitions/")
					index.schemasNode = schemasNode
					index.schemaCount = len(schemasNode.Content) / 2
				}
			}

			if n.Value == "parameters" {
				parametersNode := index.root.Content[0].Content[i+1]
				if parametersNode != nil {

					// extract params
					index.extractComponentParameters(parametersNode, "#/parameters/")
					index.parametersNode = parametersNode
				}
			}

			if n.Value == "responses" {
				responsesNode := index.root.Content[0].Content[i+1]
				if responsesNode != nil {

					// extract responses
					index.extractComponentResponses(responsesNode, "#/responses/")
					index.responsesNode = responsesNode
				}
			}

			if n.Value == "securityDefinitions" {
				securityDefinitionsNode := index.root.Content[0].Content[i+1]
				if securityDefinitionsNode != nil {

					// extract security definitions.
					index.extractComponentSecuritySchemes(securityDefinitionsNode, "#/securityDefinitions/")
					index.securitySchemesNode = securityDefinitionsNode
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

	if index.pathsNode == nil {
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

	if index.pathsNode == nil {
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
						index.scanOperationParams(params, pathItemNode, "top")
					}

					// method level params.
					if isHttpMethod(prop.Value) {

						for z, httpMethodProp := range pathPropertyNode.Content[y+1].Content {
							if z%2 == 0 {
								if httpMethodProp.Value == "parameters" {
									params := pathPropertyNode.Content[y+1].Content[z+1].Content
									index.scanOperationParams(params, pathItemNode, prop.Value)
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

								// extract description and summaries
								if httpMethodProp.Value == "description" {
									desc := pathPropertyNode.Content[y+1].Content[z+1].Value
									ref := &Reference{
										Definition: desc,
										Name:       "description",
										Node:       pathPropertyNode.Content[y+1].Content[z+1],
									}
									if index.operationDescriptionRefs[pathItemNode.Value] == nil {
										index.operationDescriptionRefs[pathItemNode.Value] = make(map[string]*Reference)
									}

									index.operationDescriptionRefs[pathItemNode.Value][prop.Value] = ref
								}
								if httpMethodProp.Value == "summary" {
									summary := pathPropertyNode.Content[y+1].Content[z+1].Value
									ref := &Reference{
										Definition: summary,
										Name:       "summary",
										Node:       pathPropertyNode.Content[y+1].Content[z+1],
									}

									if index.operationSummaryRefs[pathItemNode.Value] == nil {
										index.operationSummaryRefs[pathItemNode.Value] = make(map[string]*Reference)
									}

									index.operationSummaryRefs[pathItemNode.Value][prop.Value] = ref
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

	//now build main index of all params by combining comp refs with inline params from operations.
	//use the namespace path:::param for inline params to identify them as inline.
	for path, params := range index.paramOpRefs {
		for mName, mValue := range params {
			for pName, pValue := range mValue {
				if !strings.HasPrefix(pName, "#") {
					index.paramInlineDuplicates[pName] = append(index.paramInlineDuplicates[pName], pValue)
					index.paramAllRefs[fmt.Sprintf("%s:::%s", path, mName)] = pValue
				}
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
		located := index.FindComponent(ref.Definition, ref.Node)
		if located != nil {
			found = append(found, located)
			index.allMappedRefs[ref.Definition] = located
			index.allMappedRefsSequenced = append(index.allMappedRefsSequenced, &ReferenceMapped{
				Reference:  located,
				Definition: ref.Definition,
			})
		} else {

			_, path := utils.ConvertComponentIdIntoFriendlyPathSearch(ref.Definition)
			indexError := &IndexingError{
				Error: fmt.Errorf("component '%s' does not exist in the specification", ref.Definition),
				Node:  ref.Node,
				Path:  path,
			}
			index.refErrors = append(index.refErrors, indexError)
		}
	}
	return found
}

// FindComponent will locate a component by its reference, returns nil if nothing is found.
// This method will recurse through remote, local and file references. For each new external reference
// a new index will be created. These indexes can then be traversed recursively.
func (index *SpecIndex) FindComponent(componentId string, parent *yaml.Node) *Reference {
	if index.root == nil {
		return nil
	}

	remoteLookup := func(id string) (*yaml.Node, *yaml.Node, error) {
		return index.lookupRemoteReference(id)
	}

	fileLookup := func(id string) (*yaml.Node, *yaml.Node, error) {
		return index.lookupFileReference(id)
	}

	switch determineReferenceResolveType(componentId) {
	case localResolve: // ideally, every single ref in every single spec is local. however, this is not the case.
		return index.findComponentInRoot(componentId)

	case httpResolve:
		uri := strings.Split(componentId, "#")
		if len(uri) == 2 {
			return index.performExternalLookup(uri, componentId, remoteLookup, parent)
		}

	case fileResolve:
		uri := strings.Split(componentId, "#")
		if len(uri) == 2 {
			return index.performExternalLookup(uri, componentId, fileLookup, parent)
		}
	}
	return nil
}

// GetAllDescriptionsCount will collect together every single description found in the document
func (index *SpecIndex) GetAllDescriptionsCount() int {
	return len(index.allDescriptions)
}

// GetAllSummariesCount will collect together every single summary found in the document
func (index *SpecIndex) GetAllSummariesCount() int {
	return len(index.allSummaries)
}

/* private */

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

func (index *SpecIndex) extractDefinitionsAndSchemas(schemasNode *yaml.Node, pathPrefix string) {
	var name string
	for i, schema := range schemasNode.Content {
		if i%2 == 0 {
			name = schema.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       schema,
		}
		index.allSchemas[def] = ref
	}
}

func (index *SpecIndex) extractComponentParameters(paramsNode *yaml.Node, pathPrefix string) {
	var name string
	for i, param := range paramsNode.Content {
		if i%2 == 0 {
			name = param.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       param,
		}
		index.allParameters[def] = ref
	}
}

func (index *SpecIndex) extractComponentRequestBodies(requestBodiesNode *yaml.Node, pathPrefix string) {
	var name string
	for i, reqBod := range requestBodiesNode.Content {
		if i%2 == 0 {
			name = reqBod.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       reqBod,
		}
		index.allRequestBodies[def] = ref
	}
}

func (index *SpecIndex) extractComponentResponses(responsesNode *yaml.Node, pathPrefix string) {
	var name string
	for i, response := range responsesNode.Content {
		if i%2 == 0 {
			name = response.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       response,
		}
		index.allResponses[def] = ref
	}
}

func (index *SpecIndex) extractComponentHeaders(headersNode *yaml.Node, pathPrefix string) {
	var name string
	for i, header := range headersNode.Content {
		if i%2 == 0 {
			name = header.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       header,
		}
		index.allHeaders[def] = ref
	}
}

func (index *SpecIndex) extractComponentCallbacks(callbacksNode *yaml.Node, pathPrefix string) {
	var name string
	for i, callback := range callbacksNode.Content {
		if i%2 == 0 {
			name = callback.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       callback,
		}
		index.allCallbacks[def] = ref
	}
}

func (index *SpecIndex) extractComponentLinks(linksNode *yaml.Node, pathPrefix string) {
	var name string
	for i, link := range linksNode.Content {
		if i%2 == 0 {
			name = link.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       link,
		}
		index.allLinks[def] = ref
	}
}

func (index *SpecIndex) extractComponentExamples(examplesNode *yaml.Node, pathPrefix string) {
	var name string
	for i, example := range examplesNode.Content {
		if i%2 == 0 {
			name = example.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       example,
		}
		index.allExamples[def] = ref
	}
}

func (index *SpecIndex) extractComponentSecuritySchemes(securitySchemesNode *yaml.Node, pathPrefix string) {
	var name string
	for i, secScheme := range securitySchemesNode.Content {
		if i%2 == 0 {
			name = secScheme.Value
			continue
		}
		def := fmt.Sprintf("%s%s", pathPrefix, name)
		ref := &Reference{
			Definition: def,
			Name:       name,
			Node:       secScheme,
		}
		index.allSecuritySchemes[def] = ref
	}
}

func (index *SpecIndex) performExternalLookup(uri []string, componentId string,
	lookupFunction ExternalLookupFunction, parent *yaml.Node) *Reference {

	externalSpec := index.externalSpecIndex[uri[0]]
	var foundNode *yaml.Node
	if externalSpec == nil {

		n, newRoot, err := lookupFunction(componentId)

		if err != nil {
			indexError := &IndexingError{
				Error: err,
				Node:  parent,
				Path:  componentId,
			}
			index.refErrors = append(index.refErrors, indexError)
		}

		if n != nil {
			foundNode = n
		}

		// cool, cool, lets index this spec also. This is a recursive action and will keep going
		// until all remote references have been found.
		newIndex := NewSpecIndex(newRoot)
		index.externalSpecIndex[uri[0]] = newIndex

	} else {

		foundRef := externalSpec.findComponentInRoot(uri[1])
		if foundRef != nil {
			foundNode = foundRef.Node
		}
	}

	if foundNode != nil {
		nameSegs := strings.Split(uri[1], "/")
		ref := &Reference{
			Definition: componentId,
			Name:       nameSegs[len(nameSegs)-1],
			Node:       foundNode,
		}
		return ref
	}
	return nil
}

func (index *SpecIndex) findComponentInRoot(componentId string) *Reference {

	name, friendlySearch := utils.ConvertComponentIdIntoFriendlyPathSearch(componentId)

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

func (index *SpecIndex) scanOperationParams(params []*yaml.Node, pathItemNode *yaml.Node, method string) {
	for i, param := range params {

		// param is ref
		if len(param.Content) > 0 && param.Content[0].Value == "$ref" {

			paramRefName := param.Content[1].Value
			paramRef := index.allMappedRefs[paramRefName]

			if index.paramOpRefs[pathItemNode.Value] == nil {
				index.paramOpRefs[pathItemNode.Value] = make(map[string]map[string]*Reference)
				index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)

			}
			// if we know the path, but it's a new method
			if index.paramOpRefs[pathItemNode.Value][method] == nil {
				index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
			}
			index.paramOpRefs[pathItemNode.Value][method][paramRefName] = paramRef
			continue

		} else {

			//param is inline.
			_, vn := utils.FindKeyNode("name", param.Content)

			if vn == nil {

				path := fmt.Sprintf("$.paths.%s.%s.parameters[%d]", pathItemNode.Value, method, i)
				if method == "top" {
					path = fmt.Sprintf("$.paths.%s.parameters[%d]", pathItemNode.Value, i)
				}

				index.operationParamErrors = append(index.operationParamErrors, &IndexingError{
					Error: fmt.Errorf("the '%s' operation parameter at path '%s', index %d has no 'name' value",
						method, pathItemNode.Value, i),
					Node: param,
					Path: path,
				})
				continue
			}

			ref := &Reference{
				Definition: vn.Value,
				Name:       vn.Value,
				Node:       param,
			}
			if index.paramOpRefs[pathItemNode.Value] == nil {
				index.paramOpRefs[pathItemNode.Value] = make(map[string]map[string]*Reference)
				index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
			}

			// if we know the path but this is a new method.
			if index.paramOpRefs[pathItemNode.Value][method] == nil {
				index.paramOpRefs[pathItemNode.Value][method] = make(map[string]*Reference)
			}

			// if this is a duplicate, add an error and ignore it
			if index.paramOpRefs[pathItemNode.Value][method][ref.Name] != nil {
				path := fmt.Sprintf("$.paths.%s.%s.parameters[%d]", pathItemNode.Value, method, i)
				if method == "top" {
					path = fmt.Sprintf("$.paths.%s.parameters[%d]", pathItemNode.Value, i)
				}

				index.operationParamErrors = append(index.operationParamErrors, &IndexingError{
					Error: fmt.Errorf("the '%s' operation parameter at path '%s', index %d has a duplicate name '%s'",
						method, pathItemNode.Value, i, vn.Value),
					Node: param,
					Path: path,
				})
			} else {
				index.paramOpRefs[pathItemNode.Value][method][ref.Name] = ref
			}
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

func (index *SpecIndex) lookupRemoteReference(ref string) (*yaml.Node, *yaml.Node, error) {

	// split string to remove file reference
	uri := strings.Split(ref, "#")

	var parsedRemoteDocument *yaml.Node
	if index.seenRemoteSources[uri[0]] != nil {
		parsedRemoteDocument = index.seenRemoteSources[uri[0]]
	} else {
		resp, err := http.Get(uri[0])
		if err != nil {
			return nil, nil, err
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}

		var remoteDoc yaml.Node
		err = yaml.Unmarshal(body, &remoteDoc)
		if err != nil {
			return nil, nil, err
		}
		parsedRemoteDocument = &remoteDoc
		index.remoteLock.Lock()
		index.seenRemoteSources[uri[0]] = &remoteDoc
		index.remoteLock.Unlock()
	}

	if parsedRemoteDocument == nil {
		return nil, nil, fmt.Errorf("unable to parse remote reference: '%s'", uri[0])
	}

	// lookup item from reference by using a path query.
	query := fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))

	path, err := yamlpath.NewPath(query)
	if err != nil {
		return nil, nil, err
	}
	result, err := path.Find(parsedRemoteDocument)
	if err != nil {
		return nil, nil, err
	}
	if len(result) == 1 {
		return result[0], parsedRemoteDocument, nil
	}

	return nil, nil, nil
}

func (index *SpecIndex) lookupFileReference(ref string) (*yaml.Node, *yaml.Node, error) {

	// split string to remove file reference
	uri := strings.Split(ref, "#")

	if len(uri) != 2 {
		return nil, nil, fmt.Errorf("unable to determine filename for file reference: '%s'", ref)
	}

	file := strings.ReplaceAll(uri[0], "file:", "")

	var parsedRemoteDocument *yaml.Node
	if index.seenRemoteSources[file] != nil {
		parsedRemoteDocument = index.seenRemoteSources[file]
	} else {

		body, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, nil, err
		}

		var remoteDoc yaml.Node
		err = yaml.Unmarshal(body, &remoteDoc)
		if err != nil {
			return nil, nil, err
		}
		parsedRemoteDocument = &remoteDoc
		index.seenRemoteSources[file] = &remoteDoc
	}

	if parsedRemoteDocument == nil {
		return nil, nil, fmt.Errorf("unable to parse file reference: '%s'", file)
	}

	// lookup item from reference by using a path query.
	query := fmt.Sprintf("$%s", strings.ReplaceAll(uri[1], "/", "."))

	path, err := yamlpath.NewPath(query)
	if err != nil {
		return nil, nil, err
	}
	result, _ := path.Find(parsedRemoteDocument)
	if len(result) == 1 {
		return result[0], parsedRemoteDocument, nil
	}

	return nil, parsedRemoteDocument, nil
}
