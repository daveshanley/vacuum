// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/doctor/model/high/v3"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// schemaPathResult caches the result of a LocateSchemaPropertyPaths call.
type schemaPathResult struct {
	primaryPath string
	allPaths    []string
}

type schemaPathNodeIndexCacheKey struct {
	root *yaml.Node
}

// LocateSchemaPropertyPaths finds all paths where a schema property appears in the document.
// It uses DrDocument.LocateModelsByKeyAndValue to find all locations where the schema
// is referenced, not just its definition location.
// Results are cached in context.SchemaPathCache when available, so multiple OWASP rules
// checking the same schema avoid redundant LocateModelsByKeyAndValue calls.
// Returns the primary path and all paths where the schema appears.
func LocateSchemaPropertyPaths(
	context model.RuleFunctionContext,
	schema *v3.Schema,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	// Check cache first
	if context.SchemaPathCache != nil {
		if cached, ok := context.SchemaPathCache.Load(schema); ok {
			r := cached.(*schemaPathResult)
			return r.primaryPath, r.allPaths
		}
	}

	// Start with the schema's own path
	primaryPath = schema.GenerateJSONPath()

	lookupCompleted := false

	var locatedPaths []string

	// Try to find all locations where this schema appears
	if context.DrDocument != nil && keyNode != nil && valueNode != nil {
		locatedObjects, err := context.DrDocument.LocateModelsByKeyAndValue(keyNode, valueNode)
		if err == nil {
			lookupCompleted = true
		}
		if err == nil && locatedObjects != nil && len(locatedObjects) > 0 {
			for _, obj := range locatedObjects {
				locatedPaths = append(locatedPaths, obj.GenerateJSONPath())
			}
		}
	}

	locatedPaths = append(locatedPaths, locateSchemaReferenceAliasPaths(context, schema)...)
	if len(locatedPaths) > 0 {
		primaryPath, allPaths = buildStablePrimaryAndPaths(primaryPath, locatedPaths)

		// Store in cache
		if context.SchemaPathCache != nil {
			context.SchemaPathCache.Store(schema, &schemaPathResult{
				primaryPath: primaryPath,
				allPaths:    allPaths,
			})
		}
		return primaryPath, allPaths
	}

	// If we couldn't locate via LocateModelsByKeyAndValue,
	// fall back to the schema's own path
	primaryPath, allPaths = buildStablePrimaryAndPaths(primaryPath, nil)

	// Only cache fallback results when a full lookup actually ran.
	// This prevents nil-node calls from poisoning the cache with incomplete paths.
	if context.SchemaPathCache != nil && lookupCompleted {
		context.SchemaPathCache.Store(schema, &schemaPathResult{
			primaryPath: primaryPath,
			allPaths:    allPaths,
		})
	}
	return primaryPath, allPaths
}

func locateSchemaReferenceAliasPaths(context model.RuleFunctionContext, schema *v3.Schema) []string {
	if schema == nil || schema.Value == nil || schema.Value.GoLow() == nil ||
		schema.Value.GoLow().RootNode == nil || context.Index == nil {
		return nil
	}

	targetIndex := schema.Value.GoLow().GetIndex()
	if targetIndex == nil || targetIndex.GetRootNode() == nil {
		return nil
	}

	sourceRoot := context.Index.GetRootNode()
	if sourceRoot == nil {
		return nil
	}
	if targetIndex.GetRootNode() == sourceRoot {
		return nil
	}

	sourceDocumentPath := context.Index.GetSpecAbsolutePath()
	targetDocumentPath := targetIndex.GetSpecAbsolutePath()
	if sourceDocumentPath != "" && targetDocumentPath != "" && sourceDocumentPath == targetDocumentPath {
		return nil
	}

	targetPathIndex := nodePathIndexForSchemaContext(context, targetIndex.GetRootNode())
	targetSchemaPath, ok := targetPathIndex.Lookup(schema.Value.GoLow().RootNode)
	if !ok || targetSchemaPath == "" {
		return nil
	}

	sourcePathIndex := nodePathIndexForSchemaContext(context, sourceRoot)

	var paths []string
	for _, ref := range context.Index.GetAllSequencedReferences() {
		if ref == nil || ref.Node == nil || ref.Path == "" {
			continue
		}
		if !referenceTargetsDocument(ref, targetDocumentPath) {
			continue
		}
		sourcePath, ok := sourcePathIndex.Lookup(ref.Node)
		if !ok || sourcePath == "" {
			continue
		}
		paths = append(paths, expandSchemaReferenceAliasPaths(
			ref.Path,
			sourcePath,
			targetSchemaPath,
			targetIndex.GetAllSequencedReferences(),
			targetPathIndex,
			targetDocumentPath,
			nil,
			0,
		)...)
	}
	return paths
}

const maxSchemaReferenceAliasDepth = 16

func expandSchemaReferenceAliasPaths(
	currentTargetPath string,
	currentSourcePath string,
	targetSchemaPath string,
	targetReferences []*index.Reference,
	targetPathIndex *NodePathIndex,
	targetDocumentPath string,
	seen map[string]struct{},
	depth int,
) []string {
	if currentTargetPath == "" || currentSourcePath == "" || targetSchemaPath == "" {
		return nil
	}
	if seen == nil {
		seen = make(map[string]struct{})
	}

	seenKey := currentTargetPath + "\x00" + currentSourcePath
	if _, ok := seen[seenKey]; ok {
		return nil
	}
	seen[seenKey] = struct{}{}
	defer delete(seen, seenKey)

	var paths []string
	if suffix, ok := trimResultPathPrefix(targetSchemaPath, currentTargetPath); ok {
		paths = append(paths, canonicalizeSchemaAliasPath(currentSourcePath+suffix))
	}
	if depth >= maxSchemaReferenceAliasDepth {
		return paths
	}

	for _, nestedRef := range targetReferences {
		if nestedRef == nil || nestedRef.Node == nil || nestedRef.Path == "" {
			continue
		}
		if !referenceTargetsDocument(nestedRef, targetDocumentPath) {
			continue
		}

		nestedSourcePath, ok := targetPathIndex.Lookup(nestedRef.Node)
		if !ok || nestedSourcePath == "" {
			continue
		}
		nestedSuffix, ok := trimResultPathPrefix(nestedSourcePath, currentTargetPath)
		if !ok {
			continue
		}

		nextSourcePath := canonicalizeSchemaAliasPath(currentSourcePath + nestedSuffix)
		paths = append(paths, expandSchemaReferenceAliasPaths(
			nestedRef.Path,
			nextSourcePath,
			targetSchemaPath,
			targetReferences,
			targetPathIndex,
			targetDocumentPath,
			seen,
			depth+1,
		)...)
	}

	return paths
}

func referenceTargetsDocument(ref *index.Reference, targetDocumentPath string) bool {
	if ref == nil {
		return false
	}
	if targetDocumentPath == "" || ref.FullDefinition == "" {
		return true
	}
	if !strings.HasPrefix(ref.FullDefinition, targetDocumentPath) {
		return false
	}
	return len(ref.FullDefinition) > len(targetDocumentPath) &&
		ref.FullDefinition[len(targetDocumentPath)] == '#'
}

func trimResultPathPrefix(path, prefix string) (string, bool) {
	if resultPathHasPrefix(path, prefix) {
		return strings.TrimPrefix(path, prefix), true
	}

	normalizedPath := normalizeSimpleBracketResultPath(path)
	normalizedPrefix := normalizeSimpleBracketResultPath(prefix)
	if !resultPathHasPrefix(normalizedPath, normalizedPrefix) {
		return "", false
	}
	return strings.TrimPrefix(normalizedPath, normalizedPrefix), true
}

func normalizeSimpleBracketResultPath(path string) string {
	var b strings.Builder
	b.Grow(len(path))
	for i := 0; i < len(path); {
		if i+3 < len(path) && path[i] == '[' && (path[i+1] == '\'' || path[i+1] == '"') {
			quote := path[i+1]
			end := i + 2
			for end < len(path) && path[end] != quote {
				end++
			}
			if end+1 < len(path) && path[end+1] == ']' {
				key := path[i+2 : end]
				if IsSimpleResultPathKey(key) {
					b.WriteByte('.')
					b.WriteString(key)
					i = end + 2
					continue
				}
			}
		}
		b.WriteByte(path[i])
		i++
	}
	return b.String()
}

func canonicalizeSchemaAliasPath(path string) string {
	for _, marker := range []string{".properties.", ".patternProperties."} {
		for {
			idx := strings.Index(path, marker)
			if idx < 0 {
				break
			}
			keyStart := idx + len(marker)
			keyEnd := keyStart
			for keyEnd < len(path) && path[keyEnd] != '.' && path[keyEnd] != '[' {
				keyEnd++
			}
			if keyEnd == keyStart {
				break
			}
			key := path[keyStart:keyEnd]
			path = path[:idx+len(marker)-1] + "['" + key + "']" + path[keyEnd:]
		}
	}
	return path
}

func nodePathIndexForSchemaContext(context model.RuleFunctionContext, root *yaml.Node) *NodePathIndex {
	if root == nil {
		return nil
	}
	if context.SchemaPathCache == nil {
		return BuildNodePathIndex(root)
	}

	key := schemaPathNodeIndexCacheKey{root: root}
	if cached, ok := context.SchemaPathCache.Load(key); ok {
		if pathIndex, ok := cached.(*NodePathIndex); ok {
			return pathIndex
		}
	}

	pathIndex := BuildNodePathIndex(root)
	cached, _ := context.SchemaPathCache.LoadOrStore(key, pathIndex)
	if cachedPathIndex, ok := cached.(*NodePathIndex); ok {
		return cachedPathIndex
	}
	return pathIndex
}

func resultPathHasPrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if len(path) == len(prefix) {
		return true
	}
	next := path[len(prefix)]
	return next == '.' || next == '['
}
