// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package motor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/model/reports"
	"github.com/pb33f/jsonpath/pkg/jsonpath"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/rulesets"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/index"
)

func openLintDiagnostic(code, severity, message, path string, node *yaml.Node) model.RuleFunctionResult {
	if node == nil {
		node = &yaml.Node{Line: 1, Column: 1}
	}
	rule := &model.Rule{Id: code, Severity: severity, RuleCategory: model.RuleCategories[model.CategoryValidation]}
	return model.RuleFunctionResult{Rule: rule, RuleId: code, RuleSeverity: severity, Message: message, Path: path, StartNode: node, EndNode: vacuumUtils.BuildEndNode(node)}
}

func parseOpenLintDocument(spec []byte) (*datamodel.SpecInfo, []model.RuleFunctionResult, bool) {
	root := &yaml.Node{}
	decoder := yaml.NewDecoder(bytes.NewReader(spec))
	err := decoder.Decode(root)
	if err == io.EOF {
		root = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!null", Line: 1, Column: 1}}}
	}
	info := &datamodel.SpecInfo{RootNode: root, SpecBytes: &spec, SpecFileType: "yaml", NumLines: bytes.Count(spec, []byte{'\n'}) + 1}
	if json.Valid(spec) {
		info.SpecFileType = "json"
	}
	if err != nil && err != io.EOF {
		return info, []model.RuleFunctionResult{openLintDiagnostic("parser", "error", err.Error(), "$", nil)}, false
	}
	var results []model.RuleFunctionResult
	var next yaml.Node
	if nextErr := decoder.Decode(&next); nextErr != io.EOF {
		results = append(results, openLintDiagnostic("parser", "error", "Only one document is allowed", "$", nil))
	}
	return info, results, true
}

func openLintParserDiagnostics(root *yaml.Node, rs *rulesets.RuleSet) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	// Most documents have no parser diagnostics. Build paths only when a
	// diagnostic needs one, rather than indexing every node on every run.
	var paths *vacuumUtils.NodePathIndex
	pathFor := func(node *yaml.Node) string {
		if paths == nil {
			paths = vacuumUtils.BuildNodePathIndex(root)
		}
		path, _ := paths.Lookup(node)
		return path
	}
	var walk func(*yaml.Node)
	walk = func(n *yaml.Node) {
		if n.Kind == yaml.MappingNode {
			seen := map[string]bool{}
			for i := 0; i+1 < len(n.Content); i += 2 {
				key := n.Content[i]
				if seen[key.Value] {
					if severity := rs.ParserSeverity("duplicateKeys", "error"); severity != "off" {
						results = append(results, openLintDiagnostic("parser", severity, "Duplicate key: "+key.Value, pathFor(key), key))
					}
				}
				seen[key.Value] = true
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
					if severity := rs.ParserSeverity("incompatibleValues", "error"); severity != "off" {
						results = append(results, openLintDiagnostic("parser", severity, "Mapping keys must be strings", pathFor(key), key))
					}
				}
				walk(n.Content[i+1])
			}
		} else if n.Kind == yaml.SequenceNode || n.Kind == yaml.DocumentNode {
			for _, child := range n.Content {
				walk(child)
			}
		}
	}
	walk(root)
	return results
}

var openLintVersionPatterns = map[string]*regexp.Regexp{
	"oas2": regexp.MustCompile(`^2([^0-9]|$)`), "oas3": regexp.MustCompile(`^3([^0-9]|$)`),
	"aas2":      regexp.MustCompile(`^2\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`),
	"aas3":      regexp.MustCompile(`^3\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`),
	"arazzo1_0": regexp.MustCompile(`^1\.0(\.[0-9]*)?$`),
}

func detectOpenLintFormats(root *yaml.Node, formats []string) map[string]bool {
	found := make(map[string]bool, len(formats))
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	values := map[string]*yaml.Node{}
	if root.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(root.Content); i += 2 {
			values[root.Content[i].Value] = root.Content[i+1]
		}
	}
	text := func(key string) string {
		n := values[key]
		if n == nil {
			return ""
		}
		if n.Tag == "!!str" {
			return n.Value
		}
		var v any
		if n.Decode(&v) != nil {
			return ""
		}
		b, e := json.Marshal(v)
		if e != nil {
			return ""
		}
		return string(b)
	}
	schema := text("$schema")
	schemaNode := values["$schema"]
	isSchema := schemaNode != nil && schemaNode.Tag == "!!str" && strings.Contains(schema, "//json-schema.org/")
	isLoose := isSchema
	validType := func(n *yaml.Node) bool {
		return n != nil && n.Tag == "!!str" && strings.Contains("|array|boolean|integer|null|number|object|string|", "|"+n.Value+"|")
	}
	if n := values["type"]; n != nil {
		if n.Kind == yaml.SequenceNode {
			isLoose = true
			for _, t := range n.Content {
				if !validType(t) {
					isLoose = isSchema
					break
				}
			}
		} else if validType(n) {
			isLoose = true
		}
	}
	if n := values["enum"]; n != nil && n.Kind == yaml.SequenceNode {
		isLoose = true
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf", "not", "if"} {
		if n := values[key]; n != nil && (n.Kind == yaml.MappingNode || n.Kind == yaml.SequenceNode) {
			isLoose = true
		}
	}
	for _, format := range formats {
		matched := false
		switch format {
		case "oas2":
			matched = openLintVersionPatterns[format].MatchString(text("swagger"))
		case "oas3":
			matched = openLintVersionPatterns[format].MatchString(text("openapi"))
		case "oas3_0", "oas3_1", "oas3_2":
			matched = regexp.MustCompile(`^3\.` + format[5:] + `(\.[0-9]*)?$`).MatchString(text("openapi"))
		case "asyncapi2", "aas2":
			matched = openLintVersionPatterns["aas2"].MatchString(text("asyncapi"))
		case "aas3":
			matched = openLintVersionPatterns["aas3"].MatchString(text("asyncapi"))
		case "arazzo1_0":
			matched = openLintVersionPatterns[format].MatchString(text("arazzo"))
		case "jsonSchema":
			matched = isSchema
		case "jsonSchemaLoose":
			matched = isLoose
		default:
			if strings.HasPrefix(format, "aas2_") || strings.HasPrefix(format, "aas3_") {
				major := format[3:4]
				matched = openLintVersionPatterns[format[:4]].MatchString(text("asyncapi")) && regexp.MustCompile(`^`+major+`\.`+regexp.QuoteMeta(format[5:])+`(\.[0-9]*)?$`).MatchString(text("asyncapi"))
			}
			drafts := map[string]string{"jsonSchemaDraft4": "draft-04", "jsonSchemaDraft6": "draft-06", "jsonSchemaDraft7": "draft-07", "jsonSchemaDraft2019_09": "draft/2019-09", "jsonSchemaDraft2020_12": "draft/2020-12"}
			if draft, ok := drafts[format]; ok {
				matched = regexp.MustCompile(`^https?://json-schema\.org/` + draft + `/(hyper-)?schema#?$`).MatchString(schema)
			}
		}
		found[format] = matched
	}
	return found
}

func openLintTargets(ctx ruleContext, action model.RuleAction, node *yaml.Node) ([]*yaml.Node, error) {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if action.Field == "" {
		return []*yaml.Node{node}, nil
	}
	if action.Field == "@key" {
		if node == nil || node.Kind != yaml.MappingNode {
			return nil, nil
		}
		keys := make([]*yaml.Node, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			keys = append(keys, node.Content[i])
		}
		return keys, nil
	}
	expression := action.Field
	if expression != "$" && !strings.HasPrefix(expression, "$.") && !strings.HasPrefix(expression, "$[") {
		expression = "$." + expression
	}
	targets, err := findRuleGivenNodes(node, expression, ctx.nodeLookupTimeout, ctx.ruleJSONPathCache)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", action.Field, err)
	}
	if len(targets) == 0 {
		return []*yaml.Node{nil}, nil
	}
	return targets, nil
}

// openLintRangeIndex shares source positions across all findings in an execution.
// Columns in YAML nodes count runes; report columns count UTF-16 code units.
type openLintRangeIndex struct {
	text         []rune
	lines        []int
	wideRunes    []int
	ends         map[openLintSourceNode]int
	bounds       map[int]int
	flows        map[int]bool
	blockIndents map[int]int
}

type openLintSourceNode struct {
	offset int
	kind   yaml.Kind
}

func newOpenLintRangeIndex(source []byte, root *yaml.Node) *openLintRangeIndex {
	index := &openLintRangeIndex{text: []rune(string(source)), lines: []int{0}, ends: make(map[openLintSourceNode]int), bounds: make(map[int]int), flows: make(map[int]bool), blockIndents: make(map[int]int)}
	for offset, r := range index.text {
		if r == '\n' || (r == '\r' && (offset+1 == len(index.text) || index.text[offset+1] != '\n')) {
			index.lines = append(index.lines, offset+1)
		} else if r > 0xffff {
			index.wideRunes = append(index.wideRunes, offset)
		}
	}
	// Parser nodes are in source order. The next node bounds a scalar scan;
	// this keeps indentation and separators for later siblings out of its span.
	var offsets, boundaries []int
	linePrefixes := make(map[int]int)
	var visit func(*yaml.Node, bool, int)
	visit = func(node *yaml.Node, flow bool, parentIndent int) {
		if node == nil {
			return
		}
		offset := index.offset(node)
		if len(offsets) == 0 || offset > offsets[len(offsets)-1] {
			offsets = append(offsets, offset)
			boundary := offset
			if node.Line > 0 && node.Line <= len(index.lines) {
				lineStart := index.lines[node.Line-1]
				prefix, ok := linePrefixes[lineStart]
				if !ok {
					prefix = lineStart
					for prefix < len(index.text) {
						ch := index.text[prefix]
						if ch == ' ' || ch == '\t' {
							prefix++
							continue
						}
						if strings.ContainsRune("-?:", ch) && prefix+1 < len(index.text) && openLintSpace(index.text[prefix+1]) {
							prefix++
							continue
						}
						break
					}
					linePrefixes[lineStart] = prefix
				}
				if offset <= prefix {
					boundary = lineStart
				}
			}
			boundaries = append(boundaries, boundary)
		}
		flow = flow || node.Style&yaml.FlowStyle != 0
		if flow {
			index.flows[offset] = true
		}
		if node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
			index.blockIndents[offset] = parentIndent
		}
		if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
			parentIndent = max(0, node.Column-1)
		}
		for _, child := range node.Content {
			visit(child, flow, parentIndent)
		}
	}
	visit(root, false, 0)
	for i, offset := range offsets {
		limit := len(index.text)
		if i+1 < len(offsets) {
			limit = offsets[i+1]
			if boundaries[i+1] > offset {
				limit = boundaries[i+1]
			}
		}
		index.bounds[offset] = limit
	}
	return index
}

func (index *openLintRangeIndex) offset(node *yaml.Node) int {
	if node.Line < 1 || node.Line > len(index.lines) {
		return 0
	}
	return min(len(index.text), index.lines[node.Line-1]+max(0, node.Column-1))
}

func (index *openLintRangeIndex) position(offset int) reports.RangeItem {
	offset = min(max(0, offset), len(index.text))
	line := sort.Search(len(index.lines), func(i int) bool { return index.lines[i] > offset }) - 1
	line = max(0, line)
	start := index.lines[line]
	// Rune offsets already give the column for BMP text. Store only the
	// supplementary characters that add a second UTF-16 unit, not a column
	// integer for every character in the document. Searches stay logarithmic
	// even for compact JSON with many findings on one long line.
	column := offset - start
	if len(index.wideRunes) > 0 {
		column += sort.SearchInts(index.wideRunes, offset) - sort.SearchInts(index.wideRunes, start)
	}
	return reports.RangeItem{Line: line, Char: column}
}

// valueStart skips YAML properties; tags and anchors are not part of the value.
func (index *openLintRangeIndex) valueStart(node *yaml.Node) int {
	start := index.offset(node)
	if node.Kind != yaml.ScalarNode && node.Style&yaml.FlowStyle == 0 {
		return start
	}
	limit, ok := index.bounds[start]
	if !ok {
		limit = len(index.text)
	}
	for start < limit && (index.text[start] == '!' || index.text[start] == '&') {
		for start < limit && !openLintSpace(index.text[start]) {
			start++
		}
		propertyEnd := start
		start = index.skipTriviaTo(start, limit, false)
		if start == limit && node.Kind == yaml.ScalarNode && node.Value == "" {
			return propertyEnd
		}
	}
	return start
}

func openLintSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func (index *openLintRangeIndex) skipTrivia(offset int, comma bool) int {
	return index.skipTriviaTo(offset, len(index.text), comma)
}

func (index *openLintRangeIndex) skipTriviaTo(offset, limit int, comma bool) int {
	for offset < limit {
		ch := index.text[offset]
		if openLintSpace(ch) || (comma && ch == ',') {
			offset++
			continue
		}
		if ch != '#' {
			break
		}
		for offset < limit && index.text[offset] != '\n' && index.text[offset] != '\r' {
			offset++
		}
	}
	return offset
}

func (index *openLintRangeIndex) end(node *yaml.Node) int {
	return index.nodeEnd(node, index.flows[index.offset(node)])
}

func (index *openLintRangeIndex) nodeEnd(node *yaml.Node, flow bool) int {
	key := openLintSourceNode{offset: index.offset(node), kind: node.Kind}
	if end, ok := index.ends[key]; ok {
		return end
	}
	start := index.valueStart(node)
	end := start
	switch node.Kind {
	case yaml.DocumentNode, yaml.MappingNode, yaml.SequenceNode:
		flow = flow || node.Style&yaml.FlowStyle != 0
		for _, child := range node.Content {
			end = max(end, index.nodeEnd(child, flow))
		}
		if node.Style&yaml.FlowStyle != 0 {
			if end == start {
				end++
			}
			end = index.skipTrivia(end, true)
			if end < len(index.text) && (index.text[end] == '}' || index.text[end] == ']') {
				end++
			}
		}
	case yaml.ScalarNode:
		limit, ok := index.bounds[key.offset]
		if !ok {
			limit = len(index.text)
		}
		if start < len(index.text) && (index.text[start] == '"' || index.text[start] == '\'') {
			quote := index.text[start]
			end = start + 1
			for end < limit {
				ch := index.text[end]
				end++
				if quote == '"' && ch == '\\' {
					if end < limit {
						end++
					}
					continue
				}
				if ch == quote {
					if quote == '\'' && end < limit && index.text[end] == quote {
						end++
						continue
					}
					break
				}
			}
		} else if node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
			end = index.blockScalarEnd(start, limit, index.blockIndents[key.offset])
		} else if node.Value != "" {
			end = index.plainScalarEnd(start, limit, flow)
		}
	case yaml.AliasNode:
		end = min(len(index.text), start+1+len([]rune(node.Value)))
	}
	index.ends[key] = end
	return end
}

func (index *openLintRangeIndex) plainScalarEnd(start, limit int, flow bool) int {
	end := start
	for offset := start; offset < limit; offset++ {
		ch := index.text[offset]
		if (offset == 0 || index.text[offset-1] == '\n' || index.text[offset-1] == '\r') &&
			(ch == '-' || ch == '.') && offset+2 < limit && index.text[offset+1] == ch && index.text[offset+2] == ch &&
			(offset+3 == limit || openLintSpace(index.text[offset+3])) {
			break
		}
		if (flow && strings.ContainsRune(",[]{}", ch)) ||
			(ch == '#' && (offset == start || openLintSpace(index.text[offset-1]))) ||
			(ch == ':' && (offset+1 == len(index.text) || openLintSpace(index.text[offset+1]) || (flow && strings.ContainsRune(",[]{}", index.text[offset+1])))) {
			break
		}
		if !openLintSpace(ch) {
			end = offset + 1
		}
	}
	return end
}

func (index *openLintRangeIndex) blockScalarEnd(start, limit, baseIndent int) int {
	// A block scalar includes its source line endings, independent of the
	// decoded value's folding and chomping. The next less-indented line ends it.
	lineStart := start
	indent := 0
	offset := start + 1
	for offset < limit && index.text[offset] != '\n' && index.text[offset] != '\r' {
		if index.text[offset] >= '1' && index.text[offset] <= '9' {
			indent = baseIndent + int(index.text[offset]-'0')
		}
		if index.text[offset] == '#' { // Digits in a header comment are not indentation indicators.
			for offset < limit && index.text[offset] != '\n' && index.text[offset] != '\r' {
				offset++
			}
			break
		}
		offset++
	}
	end := offset
	for offset < limit {
		if index.text[offset] == '\r' {
			offset++
		}
		if offset < limit && index.text[offset] == '\n' {
			offset++
		}
		end = offset
		lineStart = offset
		for offset < limit && index.text[offset] == ' ' {
			offset++
		}
		if offset == limit {
			return end
		}
		if index.text[offset] != '\n' && index.text[offset] != '\r' {
			if indent == 0 {
				indent = offset - lineStart
			}
			if indent <= baseIndent || offset-lineStart < indent {
				return end
			}
		}
		for offset < limit && index.text[offset] != '\n' && index.text[offset] != '\r' {
			offset++
		}
		end = offset
	}
	return end
}

func (index *openLintRangeIndex) setResultRange(result *model.RuleFunctionResult) {
	node := result.StartNode
	if node == nil {
		return
	}
	result.Range.Start = index.position(index.valueStart(node))
	result.Range.End = index.position(index.end(node))
}

// openLintSourceLocations uses only documents already loaded by the resolver.
// External trees are parsed once from cached bytes so resolved Content pointers
// cannot change the authored bounds or paths of their owning document.
type openLintSourceLocations struct {
	root     *openLintRangeIndex
	roots    map[string]bool
	loaded   map[string]*datamodel.SpecInfo
	external map[string]*openLintExternalLocations
}

type openLintExternalLocations struct {
	ranges    *openLintRangeIndex
	locations map[openLintSourceNode]openLintNodeLocation
}

type openLintNodeLocation struct {
	node *yaml.Node
	path string
}

func newOpenLintSourceLocations(root *datamodel.SpecInfo, rootPath string, rolodexes ...*index.Rolodex) *openLintSourceLocations {
	locations := &openLintSourceLocations{
		root:     newOpenLintRangeIndex(*root.SpecBytes, root.RootNode),
		roots:    map[string]bool{"": true, rootPath: true},
		loaded:   make(map[string]*datamodel.SpecInfo),
		external: make(map[string]*openLintExternalLocations),
	}
	// Cache authored container ends before resolved nodes can supply children
	// from another source at the same root position.
	locations.root.end(root.RootNode)
	for _, rolodex := range rolodexes {
		if rolodex == nil {
			continue
		}
		if idx := rolodex.GetRootIndex(); idx != nil {
			locations.roots[idx.GetSpecAbsolutePath()] = true
		}
		for _, idx := range rolodex.GetIndexes() {
			path := idx.GetSpecAbsolutePath()
			if cfg := idx.GetConfig(); cfg != nil && cfg.SpecInfo != nil && cfg.SpecInfo.SpecBytes != nil && locations.loaded[path] == nil {
				locations.loaded[path] = cfg.SpecInfo
			}
		}
	}
	return locations
}

func (locations *openLintSourceLocations) setResultLocation(result *model.RuleFunctionResult) error {
	if result.Origin == nil || locations.roots[result.Origin.AbsoluteLocation] {
		locations.root.setResultRange(result)
		return nil
	}
	source := result.Origin.AbsoluteLocation
	external := locations.external[source]
	if external == nil {
		info := locations.loaded[source]
		if info == nil && result.Origin.Index != nil {
			if cfg := result.Origin.Index.GetConfig(); cfg != nil {
				info = cfg.SpecInfo
			}
		}
		if info == nil || info.SpecBytes == nil {
			return fmt.Errorf("source positions unavailable for loaded document %q", source)
		}
		authored, _, valid := parseOpenLintDocument(*info.SpecBytes)
		if !valid {
			return fmt.Errorf("cannot parse loaded document %q for source positions", source)
		}
		external = &openLintExternalLocations{ranges: newOpenLintRangeIndex(*info.SpecBytes, authored.RootNode), locations: make(map[openLintSourceNode]openLintNodeLocation)}
		paths := vacuumUtils.BuildNodePathIndex(authored.RootNode)
		var visit func(*yaml.Node)
		visit = func(node *yaml.Node) {
			path, _ := paths.Lookup(node)
			key := openLintSourceNode{offset: external.ranges.offset(node), kind: node.Kind}
			external.locations[key] = openLintNodeLocation{node: node, path: path}
			for _, child := range node.Content {
				visit(child)
			}
		}
		visit(authored.RootNode)
		locations.external[source] = external
	}
	if result.StartNode == nil {
		return nil
	}
	key := openLintSourceNode{offset: external.ranges.offset(result.StartNode), kind: result.StartNode.Kind}
	location, ok := external.locations[key]
	if !ok || location.node.Line != result.StartNode.Line || location.node.Column != result.StartNode.Column {
		return fmt.Errorf("source node unavailable in loaded document %q at %d:%d", source, result.StartNode.Line, result.StartNode.Column)
	}
	authoredResult := model.RuleFunctionResult{StartNode: location.node}
	external.ranges.setResultRange(&authoredResult)
	result.Range = authoredResult.Range
	result.Path = location.path
	if len(result.Paths) > 0 {
		result.Paths = []string{location.path}
	}
	result.PathFromRuleGiven = false
	return nil
}

func openLintPathSegments(path string) []string {
	p, err := jsonpath.NewPath(path)
	if err != nil || !p.IsSingular() {
		return nil
	}
	segments, err := p.GetSegmentInfo()
	if err != nil {
		return nil
	}
	parts := make([]string, 0, len(segments))
	for _, s := range segments {
		switch s.Kind {
		case jsonpath.SegmentKindMemberName:
			parts = append(parts, s.Key)
		case jsonpath.SegmentKindArrayIndex:
			parts = append(parts, strconv.FormatInt(s.Index, 10))
		}
	}
	return parts
}
