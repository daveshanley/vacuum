// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: Apache-2.0

package openapi

import (
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	drV3 "github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/datamodel"
	"go.yaml.in/yaml/v4"
)

type acceptMask uint8

const (
	acceptString acceptMask = 1 << iota
	acceptInteger
	acceptFractionalNumber
	acceptBoolean
	acceptArray
	acceptObject
	acceptNull
)

type declaredMask uint8

const (
	declaredString declaredMask = 1 << iota
	declaredNumber
	declaredInteger
	declaredBoolean
	declaredArray
	declaredObject
	declaredNull
)

// AllOfConflicts checks if properties defined across an allOf composition have incompatible type constraints.
type AllOfConflicts struct{}

// GetSchema returns the rule function schema for the allOf conflict checker.
func (a AllOfConflicts) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "allOfConflicts",
	}
}

// GetCategory returns the function category for the allOf conflict checker.
func (a AllOfConflicts) GetCategory() string {
	return model.FunctionCategoryOpenAPI
}

// RunRule walks the doctor schema graph and reports schemas whose effective allOf composition
// contains properties with no common valid type.
func (a AllOfConflicts) RunRule(_ []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if context.DrDocument == nil {
		return nil
	}

	graph := newAllOfConflictGraph(context.DrDocument.Schemas, context.SpecInfo)
	if len(graph.nodes) == 0 {
		return nil
	}
	graph.computeSCCs()
	graph.computeAggregates()

	var results []model.RuleFunctionResult
	for _, schema := range context.DrDocument.Schemas {
		if schema == nil || schema.Value == nil || len(schema.AllOf) == 0 {
			continue
		}

		nodeID, ok := graph.lookupNode(graph.canonicalize(schema))
		if !ok {
			continue
		}
		sccID := graph.nodeToSCC[nodeID]
		props := graph.effectiveAggBySCC[sccID]
		if len(props) == 0 {
			continue
		}

		var conflicting []string
		for propertyName, agg := range props {
			if agg.hasConflict() {
				conflicting = append(conflicting, propertyName)
			}
		}
		if len(conflicting) == 0 {
			continue
		}
		sort.Strings(conflicting)
		resultPath, resultPaths := a.buildAllOfPaths(schema, &context)

		for _, propertyName := range conflicting {
			result := a.buildResult(schema, propertyName, props[propertyName].declaredUnion, resultPath, resultPaths, context.Rule)
			schema.AddRuleFunctionResult(drV3.ConvertRuleResult(&result))
			results = append(results, result)
		}
	}

	return results
}

func (a AllOfConflicts) buildAllOfPaths(schema *drV3.Schema, context *model.RuleFunctionContext) (string, []string) {
	lowSchema := schema.Value.GoLow()
	locatedPath, allPaths := vacuumUtils.LocateSchemaPropertyPaths(*context, schema, lowSchema.AllOf.KeyNode, lowSchema.AllOf.ValueNode)
	path := model.GetStringTemplates().BuildJSONPath(locatedPath, "allOf")

	if len(allPaths) <= 1 {
		return path, nil
	}

	paths := make([]string, len(allPaths))
	for i, located := range allPaths {
		paths[i] = model.GetStringTemplates().BuildJSONPath(located, "allOf")
	}
	return path, paths
}

func (a AllOfConflicts) buildResult(schema *drV3.Schema, propertyName string, declared declaredMask,
	path string, paths []string, rule *model.Rule,
) model.RuleFunctionResult {
	lowSchema := schema.Value.GoLow()

	result := model.RuleFunctionResult{
		Message: fmt.Sprintf("`allOf` property `%s` declared as %s: properties defined across an `allOf` composition must have a non-empty common valid type",
			propertyName, declared.String()),
		StartNode: lowSchema.AllOf.KeyNode,
		EndNode:   vacuumUtils.BuildEndNode(lowSchema.AllOf.KeyNode),
		Path:      path,
		Rule:      rule,
	}

	if len(paths) > 0 {
		result.Paths = paths
	}

	return result
}

// declaredMask preserves user-declared schema types for messages, while acceptMask tracks
// concrete accepted values so integer and number overlap correctly during intersection.
func (d declaredMask) String() string {
	if d == 0 {
		return "[]"
	}

	var labels []string
	if d&declaredString != 0 {
		labels = append(labels, "string")
	}
	if d&declaredNumber != 0 {
		labels = append(labels, "number")
	}
	if d&declaredInteger != 0 {
		labels = append(labels, "integer")
	}
	if d&declaredBoolean != 0 {
		labels = append(labels, "boolean")
	}
	if d&declaredArray != 0 {
		labels = append(labels, "array")
	}
	if d&declaredObject != 0 {
		labels = append(labels, "object")
	}
	if d&declaredNull != 0 {
		labels = append(labels, "null")
	}
	return "[" + strings.Join(labels, ", ") + "]"
}

// propAgg tracks the merged type information for a property across one or more allOf branches.
type propAgg struct {
	declaredUnion declaredMask
	intersection  acceptMask
}

// hasConflict reports a real conflict: two or more distinct declared types with no overlap.
// Counting set bits on declaredUnion (rather than tracking contributor counts) avoids
// double-counting in diamond-shaped DAGs where the same leaf is reached via multiple paths.
func (p propAgg) hasConflict() bool {
	return p.intersection == 0 && bits.OnesCount8(uint8(p.declaredUnion)) > 1
}

type allOfConflictNode struct {
	schema     *drV3.Schema
	localProps map[string]propAgg
	children   []int
}

type allOfConflictGraph struct {
	specInfo      *datamodel.SpecInfo
	canonicalByRN map[*yaml.Node]*drV3.Schema
	nodeByRN      map[*yaml.Node]int
	nodeBySchema  map[*drV3.Schema]int
	nodes         []*allOfConflictNode

	nodeToSCC         []int
	sccMembers        [][]int
	sccChildren       [][]int
	effectiveAggBySCC []map[string]propAgg
}

func newAllOfConflictGraph(schemas []*drV3.Schema, specInfo *datamodel.SpecInfo) *allOfConflictGraph {
	graph := &allOfConflictGraph{
		specInfo:      specInfo,
		canonicalByRN: make(map[*yaml.Node]*drV3.Schema, len(schemas)),
		nodeByRN:      make(map[*yaml.Node]int, len(schemas)),
		nodeBySchema:  make(map[*drV3.Schema]int, len(schemas)),
	}

	for _, schema := range schemas {
		if root := schemaRootNode(schema); root != nil {
			if _, exists := graph.canonicalByRN[root]; !exists {
				graph.canonicalByRN[root] = schema
			}
		}
	}

	for _, schema := range schemas {
		if schema == nil || schema.Value == nil || len(schema.AllOf) == 0 {
			continue
		}
		graph.ensureNode(schema)
	}

	return graph
}

func (g *allOfConflictGraph) canonicalize(schema *drV3.Schema) *drV3.Schema {
	if schema == nil {
		return nil
	}
	if root := schemaRootNode(schema); root != nil {
		if canonical, exists := g.canonicalByRN[root]; exists && canonical != nil {
			return canonical
		}
	}
	return schema
}

func (g *allOfConflictGraph) lookupNode(schema *drV3.Schema) (int, bool) {
	if schema == nil {
		return 0, false
	}
	if root := schemaRootNode(schema); root != nil {
		id, ok := g.nodeByRN[root]
		return id, ok
	}
	id, ok := g.nodeBySchema[schema]
	return id, ok
}

func (g *allOfConflictGraph) storeNode(schema *drV3.Schema, id int) {
	if schema == nil {
		return
	}
	if root := schemaRootNode(schema); root != nil {
		g.nodeByRN[root] = id
		return
	}
	g.nodeBySchema[schema] = id
}

func (g *allOfConflictGraph) ensureNode(schema *drV3.Schema) int {
	schema = g.canonicalize(schema)
	if schema == nil || schema.Value == nil {
		return -1
	}
	if id, ok := g.lookupNode(schema); ok {
		return id
	}

	id := len(g.nodes)
	node := &allOfConflictNode{
		schema:     schema,
		localProps: extractLocalTypedProps(schema, g.specInfo),
	}
	g.nodes = append(g.nodes, node)
	g.storeNode(schema, id)

	if len(schema.AllOf) == 0 {
		return id
	}

	childSet := make(map[int]struct{}, len(schema.AllOf))
	for _, proxy := range schema.AllOf {
		if proxy == nil || proxy.Schema == nil {
			continue
		}
		childID := g.ensureNode(proxy.Schema)
		if childID < 0 {
			continue
		}
		childSet[childID] = struct{}{}
	}
	if len(childSet) == 0 {
		return id
	}

	node.children = make([]int, 0, len(childSet))
	for childID := range childSet {
		node.children = append(node.children, childID)
	}
	sort.Ints(node.children)

	return id
}

// computeSCCs collapses circular allOf components into strongly connected components so each
// recursive composition is merged once.
func (g *allOfConflictGraph) computeSCCs() {
	nodeCount := len(g.nodes)
	g.nodeToSCC = make([]int, nodeCount)
	for i := range g.nodeToSCC {
		g.nodeToSCC[i] = -1
	}
	if nodeCount == 0 {
		return
	}

	indices := make([]int, nodeCount)
	lowLink := make([]int, nodeCount)
	onStack := make([]bool, nodeCount)
	for i := range indices {
		indices[i] = -1
		lowLink[i] = -1
	}

	var stack []int
	currentIndex := 0

	var strongConnect func(int)
	strongConnect = func(v int) {
		indices[v] = currentIndex
		lowLink[v] = currentIndex
		currentIndex++
		stack = append(stack, v)
		onStack[v] = true

		for _, child := range g.nodes[v].children {
			if indices[child] == -1 {
				strongConnect(child)
				if lowLink[child] < lowLink[v] {
					lowLink[v] = lowLink[child]
				}
				continue
			}
			if onStack[child] && indices[child] < lowLink[v] {
				lowLink[v] = indices[child]
			}
		}

		if lowLink[v] != indices[v] {
			return
		}

		sccID := len(g.sccMembers)
		var members []int
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			onStack[member] = false
			g.nodeToSCC[member] = sccID
			members = append(members, member)
			if member == v {
				break
			}
		}
		g.sccMembers = append(g.sccMembers, members)
	}

	for i := 0; i < nodeCount; i++ {
		if indices[i] == -1 {
			strongConnect(i)
		}
	}

	g.sccChildren = make([][]int, len(g.sccMembers))
	for sccID, members := range g.sccMembers {
		childSet := make(map[int]struct{})
		for _, member := range members {
			for _, child := range g.nodes[member].children {
				childSCC := g.nodeToSCC[child]
				if childSCC == sccID {
					continue
				}
				childSet[childSCC] = struct{}{}
			}
		}
		if len(childSet) == 0 {
			continue
		}
		g.sccChildren[sccID] = make([]int, 0, len(childSet))
		for childSCC := range childSet {
			g.sccChildren[sccID] = append(g.sccChildren[sccID], childSCC)
		}
		sort.Ints(g.sccChildren[sccID])
	}
}

// computeAggregates folds property aggregates bottom-up across the condensed SCC DAG.
func (g *allOfConflictGraph) computeAggregates() {
	if len(g.sccMembers) == 0 {
		return
	}

	g.effectiveAggBySCC = make([]map[string]propAgg, len(g.sccMembers))
	indegree := make([]int, len(g.sccMembers))
	for _, children := range g.sccChildren {
		for _, child := range children {
			indegree[child]++
		}
	}

	queue := make([]int, 0, len(g.sccMembers))
	for sccID, degree := range indegree {
		if degree == 0 {
			queue = append(queue, sccID)
		}
	}

	topo := make([]int, 0, len(g.sccMembers))
	for head := 0; head < len(queue); head++ {
		sccID := queue[head]
		topo = append(topo, sccID)
		for _, child := range g.sccChildren[sccID] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	for i := len(topo) - 1; i >= 0; i-- {
		sccID := topo[i]
		agg := make(map[string]propAgg)
		for _, member := range g.sccMembers[sccID] {
			mergeAggMap(agg, g.nodes[member].localProps)
		}
		for _, childSCC := range g.sccChildren[sccID] {
			mergeAggMap(agg, g.effectiveAggBySCC[childSCC])
		}
		g.effectiveAggBySCC[sccID] = agg
	}
}

func mergeAggMap(dst map[string]propAgg, src map[string]propAgg) {
	for propertyName, incoming := range src {
		existing := dst[propertyName]
		dst[propertyName] = mergePropAgg(existing, incoming)
	}
}

func mergePropAgg(existing, incoming propAgg) propAgg {
	if incoming.declaredUnion == 0 {
		return existing
	}
	if existing.declaredUnion == 0 {
		return incoming
	}
	existing.declaredUnion |= incoming.declaredUnion
	existing.intersection &= incoming.intersection
	return existing
}

func extractLocalTypedProps(schema *drV3.Schema, specInfo *datamodel.SpecInfo) map[string]propAgg {
	if schema == nil || schema.Properties == nil || schema.Properties.Len() == 0 {
		return nil
	}

	props := make(map[string]propAgg, schema.Properties.Len())
	for pair := schema.Properties.First(); pair != nil; pair = pair.Next() {
		propertyProxy := pair.Value()
		if propertyProxy == nil || propertyProxy.Schema == nil {
			continue
		}

		declared, accepted, ok := normalizeSchemaTypeMasks(propertyProxy.Schema, specInfo)
		if !ok {
			continue
		}

		props[pair.Key()] = propAgg{
			declaredUnion: declared,
			intersection:  accepted,
		}
	}
	if len(props) == 0 {
		return nil
	}
	return props
}

// typeMaskTable pairs each JSON Schema type with its declared and accepted masks so the
// two stay aligned by construction; adding a new type means one entry, not two switch arms.
var typeMaskTable = map[string]struct {
	declared declaredMask
	accept   acceptMask
}{
	"string":  {declaredString, acceptString},
	"number":  {declaredNumber, acceptInteger | acceptFractionalNumber},
	"integer": {declaredInteger, acceptInteger},
	"boolean": {declaredBoolean, acceptBoolean},
	"array":   {declaredArray, acceptArray},
	"object":  {declaredObject, acceptObject},
	"null":    {declaredNull, acceptNull},
}

func normalizeSchemaTypeMasks(schema *drV3.Schema, specInfo *datamodel.SpecInfo) (declaredMask, acceptMask, bool) {
	if schema == nil || schema.Value == nil || len(schema.Value.Type) == 0 {
		return 0, 0, false
	}

	var declared declaredMask
	var accepted acceptMask
	for _, typ := range schema.Value.Type {
		if m, ok := typeMaskTable[typ]; ok {
			declared |= m.declared
			accepted |= m.accept
		}
	}

	if declared == 0 {
		return 0, 0, false
	}

	// nullable is a 3.0-only type extension; 3.1+ should express nullability via type arrays.
	if schema.Value.Nullable != nil && *schema.Value.Nullable && vacuumUtils.IsOAS30(specInfo) {
		declared |= declaredNull
		accepted |= acceptNull
	}

	return declared, accepted, true
}

func schemaRootNode(schema *drV3.Schema) *yaml.Node {
	if schema == nil || schema.Value == nil {
		return nil
	}
	lowSchema := schema.Value.GoLow()
	if lowSchema == nil {
		return nil
	}
	return lowSchema.RootNode
}
