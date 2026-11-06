package repair

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/pb33f/go-yaml"
)

// ReviewedPlan adds deterministic proposals that require explicit review to the
// built-in plan. It does not establish validity or write authority. The caller
// must validate the complete candidate and assess its contract impact.
func ReviewedPlan(source string, data []byte, findings []Finding) (Plan, error) {
	return ReviewedPlanWithSources(source, data, findings, nil)
}

// ReviewedPlanWithSources uses already frozen source documents for Arazzo
// proposals. Source keys are absolute file names or resolved URLs. It performs no I/O.
func ReviewedPlanWithSources(source string, data []byte, findings []Finding, sources map[string]*yaml.Node) (Plan, error) {
	plan, err := BuiltinPlan(source, data, findings)
	if err != nil || len(plan.KeyRemovals) > 0 {
		return plan, err
	}
	doc, err := ParseDocument(data)
	if err != nil {
		return plan, err
	}
	root := doc.Content[0]
	if err := validateTree(root, make(map[*yaml.Node]bool)); err != nil {
		return plan, nil
	}
	nodes := sourceNodes(root)
	edits := &editIndex{}
	appliedValues := make(map[string]Operation)
	for _, op := range plan.Operations {
		if op.Op == "add" || op.Op == "replace" {
			appliedValues[op.Path] = op
		}
		edits.add(operationFootprint(op, nodes))
		if op.Op == "move" || op.Op == "copy" {
			edits.add(sourceFootprint(op, nodes))
		}
	}
	for _, k := range plan.KeyTags {
		edits.add(k.Parent + "/" + EscapeToken(k.Key))
	}
	available := map[string]bool{}
	for _, t := range plan.Targets {
		available[t.RuleID+"\x00"+t.Pointer] = true
	}
	seen := map[string]bool{}
	reasons := map[string]string{}
	for _, f := range orderedFindings(findings) {
		key := f.RuleID + "\x00" + f.Pointer
		if available[key] || f.Unavailable != "" {
			continue
		}
		var ops []Operation
		var reason string
		if f.RuleID == "arazzo-source-type" || f.RuleID == "arazzo-step-operationPath" {
			ops, reason = reviewedArazzoSource(root, nodes, f, source, sources)
		} else {
			ops, reason = reviewedOperations(root, nodes, f)
		}
		if len(ops) == 0 {
			if reason != "" {
				reasons[key] = reason
			}
			continue
		}
		if plan.ReviewedProposals == nil {
			plan.ReviewedProposals = make(map[string][]Operation)
		}
		plan.ReviewedProposals[key] = ops
		remaining := make([]Operation, 0, len(ops))
		for _, op := range ops {
			previous, ok := appliedValues[op.Path]
			if ok && (op.Op == "add" || op.Op == "replace") && op.Op == previous.Op && bytes.Equal(op.Value, previous.Value) {
				continue
			}
			remaining = append(remaining, op)
		}
		ops = remaining

		encoded, _ := json.Marshal(ops)
		if !seen[string(encoded)] {
			var footprints []string
			for _, op := range ops {
				footprints = append(footprints, operationFootprint(op, nodes))
				if op.Op == "move" || op.Op == "copy" {
					footprints = append(footprints, sourceFootprint(op, nodes))
				}
			}
			conflict := false
			for _, p := range footprints {
				if edits.conflicts(p) {
					conflict = true
					break
				}
			}
			if conflict {
				reasons[key] = "repair overlaps another proposed edit"
				continue
			}
			for _, p := range footprints {
				edits.add(p)
			}
			plan.Operations = append(plan.Operations, ops...)
			for _, op := range ops {
				if op.Op == "add" || op.Op == "replace" {
					appliedValues[op.Path] = op
				}
			}

			seen[string(encoded)] = true
		}

		plan.Targets = append(plan.Targets, Target{RuleID: f.RuleID, Code: "reviewed-proposal", Pointer: f.Pointer})
		available[key] = true
	}
	declined := plan.Declined[:0]
	for _, d := range plan.Declined {
		key := d.Target.RuleID + "\x00" + d.Target.Pointer
		if available[key] {
			continue
		}
		if reason := reasons[key]; reason != "" {
			d.Reason = reason
		}
		declined = append(declined, d)
	}
	plan.Declined = declined
	return plan, nil
}

func reviewedOwner(nodes map[string]*yaml.Node, path string) (*yaml.Node, string) {
	for {
		if n := nodes[path]; n != nil && n.Kind == yaml.MappingNode {
			return n, path
		}
		if path == "" {
			return nil, ""
		}
		path = path[:strings.LastIndex(path, "/")]
	}
}
func reviewedField(nodes map[string]*yaml.Node, path, name string) (*yaml.Node, string) {
	if n, p := ancestorField(nodes, path, name); n != nil {
		return n, p
	}
	n, p := reviewedOwner(nodes, path)
	if v := field(n, name); v != nil {
		return v, p + "/" + EscapeToken(name)
	}
	return nil, ""
}
func reviewedSet(owner *yaml.Node, path, name string, value any) []Operation {
	op := "add"
	if field(owner, name) != nil {
		op = "replace"
	}
	raw, _ := json.Marshal(value)
	return []Operation{{Op: op, Path: path + "/" + EscapeToken(name), Value: raw}}
}
func reviewedOperations(root *yaml.Node, nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	n, p := reviewedOwner(nodes, f.Pointer)
	switch f.RuleID {
	case "oas3-missing-example", "oas3-valid-schema-example", "json-schema-examples-valid", "asyncapi-message-examples":
		ops, _, reason := reviewedExample(root, f, nodes)
		return ops, reason
	case "owasp-jwt-best-practices":
		format := field(n, "bearerFormat")
		if format == nil || !strings.EqualFold(format.Value, "jwt") {
			return nil, "explicit JWT bearer format required"
		}
		value := ""
		if d := field(n, "description"); d != nil {
			if d.Tag != "!!str" {
				return nil, "description must be text"
			}
			value = d.Value
		}
		if strings.Contains(value, "RFC8725") {
			return nil, "JWT reference already exists"
		}
		if value != "" {
			value += "\n\n"
		}
		value += "Implementations should follow the JWT security guidance in RFC8725 (https://www.rfc-editor.org/rfc/rfc8725)."
		return reviewedSet(n, p, "description", value), ""
	case "no-script-tags-in-markdown", "arazzo-no-script-tags-in-markdown":
		v := nodes[f.Pointer]
		if v == nil || v.Tag != "!!str" || !strings.Contains(strings.ToLower(v.Value), "<script") {
			return nil, "script markup text unavailable"
		}
		value := strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(v.Value)
		b, _ := json.Marshal(value)
		return []Operation{{Op: "replace", Path: f.Pointer, Value: b}}, ""
	case "oas3-server-trailing-slash", "asyncapi-3-server-no-trailing-slash", "asyncapi-3-channel-no-trailing-slash":
		v := nodes[f.Pointer]
		vp := f.Pointer
		if v != nil && v.Kind == yaml.MappingNode {
			key := "url"
			if f.RuleID == "asyncapi-3-channel-no-trailing-slash" {
				key = "address"
			}
			v = field(v, key)
			vp += "/" + key
		}
		if v == nil || v.Tag != "!!str" {
			return nil, "address value unavailable"
		}
		value := strings.TrimRight(v.Value, "/")
		if value == "" || value == v.Value {
			return nil, "address has no removable trailing slash"
		}
		b, _ := json.Marshal(value)
		return []Operation{{Op: "replace", Path: vp, Value: b}}, ""
	case "asyncapi-3-headers-schema-type-object":
		if n == nil || field(n, "$ref") != nil {
			return nil, "local header schema required"
		}
		if typ := field(n, "type"); typ != nil {
			return nil, "existing header type requires a contract decision"
		}
		return reviewedSet(n, p, "type", "object"), ""
	case "asyncapi-3-tags":
		info := field(root, "info")
		if info == nil {
			return nil, "info is unavailable"
		}
		tags := field(info, "tags")
		if tags != nil && (tags.Kind != yaml.SequenceNode || len(tags.Content) > 0) {
			return nil, "existing tags require individual repair"
		}
		title := field(info, "title")
		if title == nil || title.Tag != "!!str" || strings.TrimSpace(title.Value) == "" {
			return nil, "an API title is required to derive tags"
		}
		return reviewedSet(info, "/info", "tags", []map[string]string{{"name": title.Value, "description": "Operations for " + title.Value + "."}}), ""
	case "operation-tag-defined":
		tag := nodes[f.Pointer]
		parts, err := tokens(f.Pointer)
		if err != nil || len(parts) < 2 || parts[len(parts)-2] != "tags" || tag == nil || tag.Tag != "!!str" || strings.TrimSpace(tag.Value) == "" {
			return nil, "operation tag value unavailable"
		}
		global := field(root, "tags")
		if global != nil {
			if global.Kind != yaml.SequenceNode {
				return nil, "global tags are invalid"
			}
			for _, item := range global.Content {
				if name := field(item, "name"); name != nil && name.Value == tag.Value {
					return nil, "global tag already exists"
				}
			}
		}
		value := map[string]string{"name": tag.Value, "description": "Operations for " + tag.Value + "."}
		if global == nil {
			return reviewedSet(root, "", "tags", []map[string]string{value}), ""
		}
		b, _ := json.Marshal(value)
		return []Operation{{Op: "add", Path: "/tags/-", Value: b}}, ""
	case "operation-tags":
		if n == nil {
			return nil, "operation unavailable"
		}
		if tags := field(n, "tags"); tags != nil && (tags.Kind != yaml.SequenceNode || len(tags.Content) > 0) {
			return nil, "existing tags require individual repair"
		}
		parts, err := tokens(p)
		if err != nil || len(parts) != 3 || parts[0] != "paths" {
			return nil, "operation source path unavailable"
		}
		tag := ""
		for _, segment := range strings.Split(parts[1], "/") {
			if segment != "" && !strings.ContainsAny(segment, "{}") {
				tag = segment
				break
			}
		}
		if tag == "" {
			return nil, "no literal path segment establishes a tag"
		}
		global := field(root, "tags")
		found := false
		if global != nil {
			if global.Kind != yaml.SequenceNode {
				return nil, "global tags are invalid"
			}
			for _, t := range global.Content {
				if name := field(t, "name"); name != nil && name.Value == tag {
					found = true
				}
			}
		}
		ops := reviewedSet(n, p, "tags", []string{tag})
		if !found {
			value := map[string]string{"name": tag, "description": "Operations for " + tag + "."}
			if global == nil {
				ops = append(ops, reviewedSet(root, "", "tags", []map[string]string{value})...)
			} else {
				b, _ := json.Marshal(value)
				ops = append(ops, Operation{Op: "add", Path: "/tags/-", Value: b})
			}
		}
		return ops, ""

	case "oas3-example-external-check":
		external := field(n, "externalValue")
		value := field(n, "value")
		if external == nil || value == nil || external.Tag != "!!str" || !uncommented(external) || !mappingKeyUncommented(n, "externalValue") {
			return nil, "exact inline and external example values are required"
		}
		if len(external.Value) > 128<<10 {
			return nil, "embedded example exceeds proposal size limit"
		}
		var data []byte
		var err error
		if strings.HasPrefix(external.Value, "data:application/json;base64,") {
			data, err = base64.StdEncoding.DecodeString(strings.TrimPrefix(external.Value, "data:application/json;base64,"))
		} else if strings.HasPrefix(external.Value, "data:application/json,") {
			var decoded string
			decoded, err = url.PathUnescape(strings.TrimPrefix(external.Value, "data:application/json,"))
			data = []byte(decoded)
		} else {
			return nil, "external example bytes must be frozen; only embedded JSON data is available here"
		}
		if err != nil || len(data) > 64<<10 || !json.Valid(data) {
			return nil, "embedded external example is not valid bounded JSON"
		}
		document, err := ParseDocument(data)
		if err != nil || validateTree(document, make(map[*yaml.Node]bool)) != nil {
			return nil, "embedded external example is ambiguous"
		}
		equal, err := equalNodes(value, document.Content[0])
		if err != nil || !equal {
			return nil, "inline and embedded external examples differ"
		}
		return []Operation{{Op: "remove", Path: p + "/externalValue"}}, ""
	case "paths-specificity-order":
		return reviewedPathOrder(root)
	case "no-$ref-siblings", "path-item-refs":
		return reviewedLocalReference(root, nodes, f)
	case "license-url", "asyncapi-info-license-url", "info-license-spdx":
		license := field(field(root, "info"), "license")
		if license == nil {
			return nil, "license metadata is unavailable"
		}
		identifier := field(license, "identifier")
		name := field(license, "name")
		id := ""
		if identifier != nil && identifier.Tag == "!!str" {
			id = identifier.Value
		} else if name != nil && name.Tag == "!!str" {
			id = name.Value
		}
		canonical := map[string]string{"MIT": "https://opensource.org/license/mit/", "Apache-2.0": "https://www.apache.org/licenses/LICENSE-2.0", "BSD-3-Clause": "https://opensource.org/license/bsd-3-clause/"}[id]
		if canonical == "" {
			return nil, "license identity is not an exact recognized SPDX identifier"
		}
		current := field(license, "url")
		if f.RuleID == "info-license-spdx" {
			if identifier == nil || current == nil || current.Value != canonical {
				return nil, "license URL and identifier are not a proved canonical pair"
			}
			if !uncommented(current) || !mappingKeyUncommented(license, "url") {
				return nil, "license URL carries comments"
			}
			return []Operation{{Op: "remove", Path: "/info/license/url"}}, ""
		}
		if current != nil && current.Value != "" {
			return nil, "license URL already exists"
		}
		return reviewedSet(license, "/info/license", "url", canonical), ""
	case "oas2-schema", "oas3-schema", "json-schema-valid", "arazzo-structure":
		return reviewedStructuralDuplicates(nodes, f)
	case "json-schema-type-constraint-compatible":
		return reviewedInertConstraint(nodes, f)
	case "arazzo-advisory", "arazzo-duplicate-id":
		return reviewedArazzoIdentifier(nodes, f)
	case "asyncapi-latest-version":
		return nil, "a validated version-specific AsyncAPI migration is required; changing only the version label is not a repair"
	case "migrate-zally-ignore":
		legacy := nodes[f.Pointer]
		if !strings.HasSuffix(f.Pointer, "/x-zally-ignore") || legacy == nil || legacy.Kind != yaml.SequenceNode || len(legacy.Content) != 0 {
			return nil, "nonempty ignore migration requires a verified rule-ID mapping and suppression-equivalence check"
		}
		ownerPath := f.Pointer[:strings.LastIndex(f.Pointer, "/")]
		owner := nodes[ownerPath]
		if field(owner, "x-lint-ignore") != nil {
			if !uncommented(legacy) || !mappingKeyUncommented(owner, "x-zally-ignore") {
				return nil, "legacy ignore carries comments"
			}
			return []Operation{{Op: "remove", Path: f.Pointer}}, ""
		}
		return []Operation{{Op: "move", From: f.Pointer, Path: ownerPath + "/x-lint-ignore"}}, ""
	case "no-eval-in-markdown":
		v := nodes[f.Pointer]
		if v == nil || v.Tag != "!!str" || !strings.Contains(v.Value, "eval(") {
			return nil, "literal eval expression unavailable"
		}
		value := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "eval(", "eval&#40;").Replace(v.Value)
		b, _ := json.Marshal(value)
		return []Operation{{Op: "replace", Path: f.Pointer, Value: b}}, ""
	case "oas2-unused-definition", "oas3-unused-component", "asyncapi-unused-components":
		parts, _ := tokens(f.Pointer)
		schema := len(parts) == 2 && parts[0] == "definitions" || len(parts) == 3 && parts[0] == "components" && parts[1] == "schemas"
		if !schema || nodes[f.Pointer] == nil {
			return nil, "unused removal currently requires a root-local schema"
		}
		for path, node := range nodes {
			if strings.HasSuffix(path, "/discriminator") || strings.HasSuffix(path, "/$id") || strings.HasSuffix(path, "/$dynamicRef") || strings.HasSuffix(path, "/$recursiveRef") {
				return nil, "schema has scoped or implicit reference semantics"
			}
			if strings.HasSuffix(path, "/$ref") {
				ref, err := url.PathUnescape(node.Value)
				if err != nil || !strings.HasPrefix(ref, "#/") {
					return nil, "unused removal requires a closed local pointer-reference graph"
				}
				target := strings.TrimPrefix(ref, "#")
				if target == f.Pointer || strings.HasPrefix(target, f.Pointer+"/") {
					return nil, "schema has an incoming reference"
				}
			}
		}
		if !uncommented(nodes[f.Pointer]) {
			return nil, "unused schema carries comments"
		}
		return []Operation{{Op: "remove", Path: f.Pointer}}, ""
	case "asyncapi-channel-parameters":
		address := field(n, "address")
		if address == nil || address.Tag != "!!str" {
			return nil, "channel address required"
		}
		parameters := field(n, "parameters")
		if parameters != nil && parameters.Kind != yaml.MappingNode {
			return nil, "channel parameters must be an object"
		}
		names := map[string]bool{}
		for _, match := range reviewedTemplateVariable.FindAllStringSubmatch(address.Value, -1) {
			names[match[1]] = true
		}
		if parameters != nil {
			for i := 0; i < len(parameters.Content); i += 2 {
				if !names[parameters.Content[i].Value] {
					return nil, "unused authored parameters require a separate removal decision"
				}
			}
		}
		sorted := make([]string, 0, len(names))
		for name := range names {
			sorted = append(sorted, name)
		}
		sort.Strings(sorted)
		var ops []Operation
		values := map[string]any{}
		for _, name := range sorted {
			if field(parameters, name) != nil {
				continue
			}
			value := map[string]string{"description": "Value for the " + name + " channel address parameter."}
			values[name] = value
			if parameters != nil {
				b, _ := json.Marshal(value)
				ops = append(ops, Operation{Op: "add", Path: p + "/parameters/" + EscapeToken(name), Value: b})
			}
		}
		if parameters == nil && len(values) > 0 {
			ops = reviewedSet(n, p, "parameters", values)
		}
		return ops, "all address parameters are declared"
	case "required-fields-defined":
		required, rp := reviewedField(nodes, f.Pointer, "required")
		if required == nil || required.Kind != yaml.SequenceNode {
			return nil, "required array unavailable"
		}
		schemaPath := rp[:strings.LastIndex(rp, "/")]
		schema := nodes[schemaPath]
		properties := field(schema, "properties")
		example := field(schema, "example")
		if example == nil || example.Kind != yaml.MappingNode {
			return nil, "an existing object example is required to derive missing properties"
		}
		if properties != nil && properties.Kind != yaml.MappingNode {
			return nil, "properties must be an object"
		}
		values := map[string]any{}
		var ops []Operation
		for _, name := range required.Content {
			if name.Tag != "!!str" {
				return nil, "required names must be strings"
			}
			if field(properties, name.Value) != nil {
				continue
			}
			sample := field(example, name.Value)
			typ := ""
			if sample != nil {
				switch sample.Tag {
				case "!!str":
					typ = "string"
				case "!!int":
					typ = "integer"
				case "!!float":
					typ = "number"
				case "!!bool":
					typ = "boolean"
				}
			}
			if typ == "" {
				return nil, "missing property needs an unambiguous scalar example"
			}
			value := map[string]string{"type": typ}
			if _, ok := values[name.Value]; ok {
				continue
			}
			values[name.Value] = value
			if properties != nil {
				b, _ := json.Marshal(value)
				ops = append(ops, Operation{Op: "add", Path: schemaPath + "/properties/" + EscapeToken(name.Value), Value: b})
			}
		}
		if properties == nil && len(values) > 0 {
			ops = reviewedSet(schema, schemaPath, "properties", values)
		}
		return ops, "required properties are already declared"
	case "camel-case-properties":
		return reviewedPropertyRename(root, nodes, f)
	case "operation-operationId-valid-in-url", "operation-operationId-unique":
		return reviewedOperationID(root, nodes, f)
	case "paths-kebab-case", "path-keys-no-trailing-slash", "no-http-verbs-in-path", "path-not-include-query":
		return reviewedPathRename(root, nodes, f)
	case "asyncapi-3-channel-no-query-nor-fragment":
		v := nodes[f.Pointer]
		vp := f.Pointer
		if v != nil && v.Kind == yaml.MappingNode {
			v = field(v, "address")
			vp += "/address"
		}
		if v == nil || v.Tag != "!!str" {
			return nil, "channel address unavailable"
		}
		i := strings.IndexAny(v.Value, "?#")
		if i < 0 || i == 0 {
			return nil, "no nonempty channel address can be retained"
		}
		b, _ := json.Marshal(v.Value[:i])
		return []Operation{{Op: "replace", Path: vp, Value: b}}, ""
	case "json-schema-title-description-type":
		if n == nil || field(n, "type") == nil {
			return nil, "quality proposal requires an existing type"
		}
		typ := field(n, "type")
		if typ.Value == "object" && field(n, "properties") == nil && field(n, "additionalProperties") == nil && field(n, "unevaluatedProperties") == nil {
			return nil, "object constraints need an intended shape"
		}
		if typ.Value == "array" && field(n, "items") == nil && field(n, "prefixItems") == nil && field(n, "contains") == nil {
			return nil, "array items need an intended shape"
		}
		parts, _ := tokens(p)
		label := "Document"
		if len(parts) > 0 {
			label = parts[len(parts)-1]
		}
		var ops []Operation
		for _, key := range []string{"title", "description"} {
			v := field(n, key)
			if v != nil && v.Tag != "!!str" {
				return nil, "metadata must be text"
			}
			if v == nil || strings.TrimSpace(v.Value) == "" {
				value := label
				if key == "description" {
					value = "Schema for " + label + "."
				}
				ops = append(ops, reviewedSet(n, p, key, value)...)
			}
		}
		return ops, "metadata already exists"
	case "json-schema-composition-sanity":
		seq, sp := reviewedField(nodes, f.Pointer, "oneOf")
		if seq == nil || seq.Kind != yaml.SequenceNode {
			return nil, "only identical oneOf branches have a generated proposal"
		}
		seen := map[string]bool{}
		var remove []int
		for i, v := range seq.Content {
			b, err := raw(v)
			if err != nil {
				return nil, "branch cannot be compared"
			}
			var value any
			decoder := json.NewDecoder(bytes.NewReader(b))
			decoder.UseNumber()
			if decoder.Decode(&value) != nil {
				return nil, "branch cannot be compared"
			}
			canonical, _ := json.Marshal(value)
			key := string(canonical)
			if seen[key] {
				if !uncommented(v) {
					return nil, "duplicate branch carries comments"
				}
				remove = append(remove, i)
			} else {
				seen[key] = true
			}
		}
		var ops []Operation
		for i := len(remove) - 1; i >= 0; i-- {
			ops = append(ops, Operation{Op: "remove", Path: sp + "/" + strconv.Itoa(remove[i])})
		}
		return ops, "no duplicate branches"
	case "oas2-discriminator":
		d := field(n, "discriminator")
		if d == nil || d.Tag != "!!str" || strings.TrimSpace(d.Value) == "" {
			return nil, "a named string discriminator is required"
		}
		var ops []Operation
		required := field(n, "required")
		if required == nil {
			ops = append(ops, reviewedSet(n, p, "required", []string{d.Value})...)
		} else {
			if required.Kind != yaml.SequenceNode {
				return nil, "required must be an array"
			}
			present := false
			for _, v := range required.Content {
				present = present || v.Value == d.Value
			}
			if !present {
				b, _ := json.Marshal(d.Value)
				ops = append(ops, Operation{Op: "add", Path: p + "/required/-", Value: b})
			}
		}
		properties := field(n, "properties")
		if properties == nil {
			ops = append(ops, reviewedSet(n, p, "properties", map[string]any{d.Value: map[string]string{"type": "string"}})...)
		} else {
			if properties.Kind != yaml.MappingNode {
				return nil, "properties must be an object"
			}
			property := field(properties, d.Value)
			if property == nil {
				b, _ := json.Marshal(map[string]string{"type": "string"})
				ops = append(ops, Operation{Op: "add", Path: p + "/properties/" + EscapeToken(d.Value), Value: b})
			} else {
				typ := field(property, "type")
				if typ == nil || typ.Value != "string" {
					return nil, "existing discriminator property requires a type decision"
				}
			}
		}
		return ops, "discriminator already declared"
	case "oas2-operation-formData-consume-check":
		parts, err := tokens(f.Pointer)
		if err != nil || len(parts) < 5 || parts[0] != "paths" || parts[3] != "parameters" {
			return nil, "local operation form parameter required"
		}
		opPath := "/paths/" + EscapeToken(parts[1]) + "/" + EscapeToken(parts[2])
		operation := nodes[opPath]
		params := field(operation, "parameters")
		if params == nil || params.Kind != yaml.SequenceNode {
			return nil, "operation parameters unavailable"
		}
		media := "application/x-www-form-urlencoded"
		found := false
		for _, param := range params.Content {
			if field(param, "$ref") != nil {
				return nil, "referenced parameters require resolved encoding analysis"
			}
			in := field(param, "in")
			if in != nil && in.Value == "formData" {
				found = true
				if typ := field(param, "type"); typ != nil && typ.Value == "file" {
					media = "multipart/form-data"
				}
			}
		}
		if !found {
			return nil, "operation has no local form parameters"
		}
		consumes := field(operation, "consumes")
		if consumes == nil {
			return reviewedSet(operation, opPath, "consumes", []string{media}), ""
		}
		if consumes.Kind != yaml.SequenceNode {
			return nil, "consumes must be an array"
		}
		for _, v := range consumes.Content {
			if v.Value == media {
				return nil, "form encoding already declared"
			}
		}
		b, _ := json.Marshal(media)
		return []Operation{{Op: "add", Path: opPath + "/consumes/-", Value: b}}, ""
	case "nullable-enum-contains-null":
		enum, ep := reviewedField(nodes, f.Pointer, "enum")
		if enum == nil || enum.Kind != yaml.SequenceNode {
			return nil, "enum array unavailable"
		}
		owner := nodes[ep[:strings.LastIndex(ep, "/")]]
		nullable := field(owner, "nullable")
		allows := nullable != nil && nullable.Tag == "!!bool" && nullable.Value == "true"
		typ := field(owner, "type")
		if typ != nil && typ.Kind == yaml.SequenceNode {
			for _, t := range typ.Content {
				allows = allows || (t.Tag == "!!str" && t.Value == "null")
			}
		}
		if !allows {
			return nil, "schema does not explicitly permit null"
		}
		for _, v := range enum.Content {
			if v.Tag == "!!null" {
				return nil, "enum already contains null"
			}
		}
		return []Operation{{Op: "add", Path: ep + "/-", Value: json.RawMessage("null")}}, ""
	case "json-schema-required-properties-defined", "operation-singular-tag":
		name := "required"
		if f.RuleID == "operation-singular-tag" {
			name = "tags"
		}
		seq, sp := reviewedField(nodes, f.Pointer, name)
		if seq == nil || seq.Kind != yaml.SequenceNode {
			return nil, "string array unavailable"
		}
		first := map[string]int{}
		var remove []int
		for i, v := range seq.Content {
			if v.Tag != "!!str" {
				return nil, "array contains a non-string value"
			}
			if _, ok := first[v.Value]; ok {
				if !uncommented(v) {
					return nil, "duplicate carries comments"
				}
				remove = append(remove, i)
			} else {
				first[v.Value] = i
			}
		}
		if name == "tags" && len(first) != 1 {
			return nil, "distinct tags require a primary-tag decision"
		}
		var ops []Operation
		for i := len(remove) - 1; i >= 0; i-- {
			ops = append(ops, Operation{Op: "remove", Path: sp + "/" + strconv.Itoa(remove[i])})
		}
		return ops, "no duplicate entries"
	case "oas2-anyOf", "oas2-oneOf":
		key := "anyOf"
		if f.RuleID == "oas2-oneOf" {
			key = "oneOf"
		}
		seq, sp := reviewedField(nodes, f.Pointer, key)
		if seq == nil || seq.Kind != yaml.SequenceNode || len(seq.Content) != 1 {
			return nil, "only a singleton composition has an equivalent conversion"
		}
		owner := nodes[sp[:strings.LastIndex(sp, "/")]]
		version := field(root, "swagger")
		if version == nil || version.Value != "2.0" || field(owner, "allOf") != nil {
			return nil, "conversion requires Swagger 2.0 and no existing allOf"
		}
		return []Operation{{Op: "move", From: sp, Path: sp[:strings.LastIndex(sp, "/")] + "/allOf"}}, ""
	case "oas2-host-trailing-slash":
		host := field(root, "host")
		if host == nil || host.Tag != "!!str" {
			return nil, "host unavailable"
		}
		v := strings.TrimRight(host.Value, "/")
		u, err := url.Parse("http://" + v)
		if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || v == host.Value {
			return nil, "host must contain only an authority and trailing slashes"
		}
		return reviewedSet(root, "", "host", v), ""
	case "oas-missing-type":
		if n == nil || field(n, "type") != nil || field(n, "$ref") != nil {
			return nil, "type is already declared or referenced"
		}
		candidate := ""
		for _, g := range []struct {
			typ  string
			keys []string
		}{{"number", []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}}, {"string", []string{"minLength", "maxLength", "pattern"}}, {"array", []string{"minItems", "maxItems", "uniqueItems"}}, {"object", []string{"minProperties", "maxProperties", "required"}}} {
			for _, k := range g.keys {
				if field(n, k) != nil {
					if candidate != "" && candidate != g.typ {
						return nil, "keywords imply multiple types"
					}
					candidate = g.typ
				}
			}
		}
		if candidate == "" {
			return nil, "no declared constraints establish a proposed type"
		}
		return reviewedSet(n, p, "type", candidate), ""
	case "typed-enum":
		enum, ep := reviewedField(nodes, f.Pointer, "enum")
		if enum == nil || enum.Kind != yaml.SequenceNode {
			return nil, "enum unavailable"
		}
		owner := nodes[ep[:strings.LastIndex(ep, "/")]]
		typ := field(owner, "type")
		if typ == nil || typ.Kind != yaml.ScalarNode {
			return nil, "one declared type is required"
		}
		var ops []Operation
		for i, v := range enum.Content {
			var value any
			switch typ.Value {
			case "integer":
				if v.Tag == "!!int" {
					continue
				}
				if v.Tag != "!!str" {
					return nil, "enum conversion is not an exact string conversion"
				}
				x, err := strconv.ParseInt(v.Value, 10, 64)
				if err != nil || strconv.FormatInt(x, 10) != v.Value {
					return nil, "integer spelling is ambiguous or out of range"
				}
				value = x
			case "boolean":
				if v.Tag == "!!bool" {
					continue
				}
				if v.Tag != "!!str" || (v.Value != "true" && v.Value != "false") {
					return nil, "boolean spelling is ambiguous"
				}
				value = v.Value == "true"
			default:
				return nil, "this enum type has no lossless conversion recipe"
			}
			b, _ := json.Marshal(value)
			ops = append(ops, Operation{Op: "replace", Path: ep + "/" + strconv.Itoa(i), Value: b})
		}
		return ops, "enum values already have their declared type"
	case "owasp-integer-format", "owasp-integer-limit":
		typ := field(n, "type")
		if typ == nil || typ.Value != "integer" {
			return nil, "one explicit integer type is required"
		}
		if f.RuleID == "owasp-integer-format" {
			min, max := field(n, "minimum"), field(n, "maximum")
			if min == nil || max == nil {
				return nil, "both integer bounds are required to infer format"
			}
			lo, ok := new(big.Int).SetString(min.Value, 10)
			if !ok {
				return nil, "minimum is not an integer"
			}
			hi, ok := new(big.Int).SetString(max.Value, 10)
			if !ok || lo.Cmp(hi) > 0 {
				return nil, "invalid integer range"
			}
			format := "int64"
			if lo.IsInt64() && hi.IsInt64() {
				if lo.Int64() >= -2147483648 && hi.Int64() <= 2147483647 {
					format = "int32"
				}
			} else {
				return nil, "range exceeds int64"
			}
			if field(n, "format") != nil {
				return nil, "existing format requires a format decision"
			}
			return reviewedSet(n, p, "format", format), ""
		}
		format := field(n, "format")
		if format == nil || (format.Value != "int32" && format.Value != "int64") {
			return nil, "an explicit integer width is required"
		}
		lo, hi := int64(-2147483648), int64(2147483647)
		if format.Value == "int64" {
			lo, hi = -9223372036854775808, 9223372036854775807
		}
		if field(n, "exclusiveMinimum") != nil || field(n, "exclusiveMaximum") != nil {
			return nil, "exclusive bounds require separate review"
		}
		// Existing inclusive bounds must fit the declared width.
		for _, key := range []string{"minimum", "maximum"} {
			if v := field(n, key); v != nil {
				x, err := strconv.ParseInt(v.Value, 10, 64)
				if err != nil || x < lo || x > hi {
					return nil, "existing bounds conflict with declared width"
				}
			}
		}
		var ops []Operation
		if field(n, "minimum") == nil {
			ops = append(ops, reviewedSet(n, p, "minimum", lo)...)
		}
		if field(n, "maximum") == nil {
			ops = append(ops, reviewedSet(n, p, "maximum", hi)...)
		}
		return ops, "integer bounds already exist"
	case "owasp-auth-urls-https", "owasp-security-hosts-https-oas3":
		v := nodes[f.Pointer]
		vp := f.Pointer
		if v != nil && v.Kind == yaml.MappingNode {
			v = field(v, "url")
			vp += "/url"
		}
		if v == nil || v.Tag != "!!str" {
			return nil, "URL value unavailable"
		}
		u, err := url.Parse(v.Value)
		if err != nil || u.Scheme != "http" || u.Host == "" || u.User != nil || u.Port() != "" {
			return nil, "only plain HTTP URLs without credentials or explicit ports have a proposal"
		}
		u.Scheme = "https"
		b, _ := json.Marshal(u.String())
		return []Operation{{Op: "replace", Path: vp, Value: b}}, ""
	case "owasp-no-additionalProperties":
		if n == nil || field(n, "type") == nil || field(n, "type").Value != "object" {
			return nil, "explicit object schema required"
		}
		v := field(n, "additionalProperties")
		if v != nil && (v.Tag != "!!bool" || v.Value != "true") {
			return nil, "authored additional-property schema requires a separate decision"
		}
		return reviewedSet(n, p, "additionalProperties", false), ""
	case "description-duplication":
		v := nodes[f.Pointer]
		if v == nil || v.Tag != "!!str" || (f.Code != "description-duplicate" && f.Code != "summary-duplicate") {
			return nil, "exact duplicate text diagnostic required"
		}
		parts, err := tokens(p)
		if err != nil || len(parts) == 0 {
			return nil, "description has no named owner"
		}
		label := reviewedDescriptionLabel(parts, n)
		b, _ := json.Marshal(v.Value + " (" + label + ")")
		return []Operation{{Op: "replace", Path: f.Pointer, Value: b}}, ""
	}
	return nil, ""
}

// ReviewedRuleIDs lists rules with additional review-only recipes. Availability
// still depends on the diagnostic, source shape and guards in ReviewedPlan.
func ReviewedRuleIDs() []string {
	return []string{"oas3-missing-example", "oas3-valid-schema-example", "json-schema-examples-valid", "asyncapi-message-examples", "arazzo-source-type", "arazzo-step-operationPath", "oas3-example-external-check", "arazzo-advisory", "arazzo-duplicate-id", "paths-specificity-order", "no-$ref-siblings", "path-item-refs", "migrate-zally-ignore", "license-url", "asyncapi-info-license-url", "info-license-spdx", "oas2-schema", "oas3-schema", "json-schema-valid", "arazzo-structure", "json-schema-type-constraint-compatible", "no-eval-in-markdown", "oas2-unused-definition", "oas3-unused-component", "asyncapi-unused-components", "asyncapi-channel-parameters", "required-fields-defined", "camel-case-properties", "operation-operationId-valid-in-url", "operation-operationId-unique", "paths-kebab-case", "path-keys-no-trailing-slash", "no-http-verbs-in-path", "path-not-include-query", "asyncapi-3-channel-no-query-nor-fragment", "json-schema-title-description-type", "json-schema-composition-sanity", "oas2-discriminator", "oas2-operation-formData-consume-check", "owasp-jwt-best-practices", "no-script-tags-in-markdown", "arazzo-no-script-tags-in-markdown", "oas3-server-trailing-slash", "asyncapi-3-server-no-trailing-slash", "asyncapi-3-channel-no-trailing-slash", "asyncapi-3-headers-schema-type-object", "asyncapi-3-tags", "operation-tags", "operation-tag-defined", "nullable-enum-contains-null", "json-schema-required-properties-defined", "operation-singular-tag", "oas2-anyOf", "oas2-oneOf", "oas2-host-trailing-slash", "oas-missing-type", "typed-enum", "owasp-integer-format", "owasp-integer-limit", "owasp-auth-urls-https", "owasp-security-hosts-https-oas3", "owasp-no-additionalProperties", "description-duplication"}
}

func reviewedIdentifier(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func reviewedOperationID(root *yaml.Node, nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	owner, p := reviewedOwner(nodes, f.Pointer)
	id := field(owner, "operationId")
	parts, _ := tokens(p)
	if id == nil || id.Tag != "!!str" || len(parts) != 3 || parts[0] != "paths" {
		return nil, "local path operation identifier required"
	}
	var pointers []string
	ids := map[string]bool{}
	definitions := map[string]bool{}
	occurrences := 0
	for path, node := range nodes {
		if !strings.HasSuffix(path, "/operationId") || node.Tag != "!!str" {
			continue
		}
		ps, _ := tokens(path)
		definition := len(ps) == 4 && ps[0] == "paths" && reviewedHTTPMethod(ps[2])
		link := len(ps) == 4 && ps[0] == "components" && ps[1] == "links" || len(ps) == 6 && ps[0] == "components" && ps[1] == "responses" && ps[3] == "links" || len(ps) == 8 && (ps[0] == "paths" || ps[0] == "webhooks") && reviewedHTTPMethod(ps[2]) && ps[3] == "responses" && ps[5] == "links"
		if !definition && !link && reviewedDataPath(ps) {
			continue
		}
		if !definition && !link {
			return nil, "unrecognized operationId owner requires reference analysis"
		}
		if definition {
			ids[node.Value] = true
			definitions[path] = true
			if node.Value == id.Value {
				occurrences++
			}
		}
		if node.Value == id.Value {
			pointers = append(pointers, path)
		}
	}
	sort.Strings(pointers)
	if f.RuleID == "operation-operationId-unique" {
		for _, path := range pointers {
			if !definitions[path] {
				return nil, "duplicate operation identifier has ambiguous references"
			}
		}
	} else if occurrences != 1 {
		return nil, "identifier normalization requires an unambiguous definition"
	}
	value := reviewedIdentifier(id.Value)
	if f.RuleID == "operation-operationId-unique" || value == "" {
		value = reviewedIdentifier(parts[2] + "_" + parts[1])
	}
	if value == id.Value && f.RuleID != "operation-operationId-unique" {
		return nil, "identifier already uses supported characters"
	}
	base := value
	for i := 2; ids[value]; i++ {
		value = base + "_" + strconv.Itoa(i)
	}
	b, _ := json.Marshal(value)
	ops := []Operation{{Op: "replace", Path: p + "/operationId", Value: b}}
	if f.RuleID != "operation-operationId-unique" {
		for _, path := range pointers {
			if path != p+"/operationId" {
				ops = append(ops, Operation{Op: "replace", Path: path, Value: b})
			}
		}
	}
	return ops, ""
}

func reviewedPathRename(root *yaml.Node, nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	parts, _ := tokens(f.Pointer)
	if len(parts) != 2 || parts[0] != "paths" {
		return nil, "path item source required"
	}
	for path, node := range nodes {
		if strings.HasSuffix(path, "/$ref") || strings.HasSuffix(path, "/$dynamicRef") || strings.HasSuffix(path, "/$recursiveRef") || strings.HasSuffix(path, "/operationRef") {
			if node.Value != "" {
				return nil, "path renaming requires a closed reference-free document"
			}
		}
	}
	old := parts[1]
	value := old
	switch f.RuleID {
	case "path-keys-no-trailing-slash":
		value = strings.TrimRight(old, "/")
		if value == "" {
			return nil, "root path cannot be removed"
		}
	case "path-not-include-query":
		if !strings.HasSuffix(old, "?") || strings.Count(old, "?") != 1 {
			return nil, "nonempty query parameters require a parameter migration"
		}
		value = strings.TrimSuffix(old, "?")
	case "no-http-verbs-in-path":
		segments := strings.Split(old, "/")
		kept := []string{}
		for _, segment := range segments {
			switch strings.ToLower(segment) {
			case "get", "post", "put", "patch", "delete", "head", "options", "trace", "connect":
				continue
			}
			kept = append(kept, segment)
		}
		value = strings.Join(kept, "/")
		if value == "" {
			value = "/"
		}
	case "paths-kebab-case":
		segments := strings.Split(old, "/")
		for i, segment := range segments {
			if strings.ContainsAny(segment, "{}") {
				continue
			}
			var b strings.Builder
			for j, r := range segment {
				if r > 127 {
					return nil, "non-ASCII paths need an explicit naming policy"
				}
				if r >= 'A' && r <= 'Z' {
					if j > 0 && b.Len() > 0 {
						b.WriteByte('-')
					}
					b.WriteRune(unicode.ToLower(r))
				} else if r == '_' || r == ' ' {
					b.WriteByte('-')
				} else {
					b.WriteRune(r)
				}
			}
			segments[i] = b.String()
		}
		value = strings.Join(segments, "/")
	}
	if value == old || value == "" {
		return nil, "path has no supported normalization"
	}
	paths := field(root, "paths")
	if field(paths, value) != nil {
		return nil, "normalized path already exists"
	}
	return []Operation{{Op: "move", From: f.Pointer, Path: "/paths/" + EscapeToken(value)}}, ""
}

var reviewedTemplateVariable = regexp.MustCompile(`\{([^{}]+)\}`)

func reviewedPropertyRename(root *yaml.Node, nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	parts, _ := tokens(f.Pointer)
	if len(parts) < 2 || parts[len(parts)-2] != "properties" {
		return nil, "schema property source required"
	}
	for path, node := range nodes {
		if (strings.HasSuffix(path, "/$ref") || strings.HasSuffix(path, "/$dynamicRef") || strings.HasSuffix(path, "/$recursiveRef")) && node.Value != "" {
			return nil, "property renaming requires a closed reference-free document"
		}
	}
	old := parts[len(parts)-1]
	words := strings.FieldsFunc(old, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	if len(words) == 0 {
		return nil, "property name has no words"
	}
	value := ""
	for i, word := range words {
		if word == "" {
			continue
		}
		for _, r := range word {
			if r > 127 || !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return nil, "property spelling needs an explicit naming policy"
			}
		}
		if i == 0 {
			value = strings.ToLower(word[:1]) + word[1:]
		} else {
			value += strings.ToUpper(word[:1]) + word[1:]
		}
	}
	if value == old || value == "" || value[0] < 'a' || value[0] > 'z' {
		return nil, "property has no supported camel-case conversion"
	}
	parentPath := f.Pointer[:strings.LastIndex(f.Pointer, "/")]
	properties := nodes[parentPath]
	if field(properties, value) != nil {
		return nil, "normalized property already exists"
	}
	schemaPath := parentPath[:strings.LastIndex(parentPath, "/")]
	schema := nodes[schemaPath]
	// A nested rename also changes instances described by its ancestors. Only
	// the immediate example is migrated below; other authored data or dependent
	// constraints need a coordinated migration, not a local key rename.
	for path := schemaPath; ; {
		for _, key := range []string{"dependentRequired", "dependentSchemas", "dependencies", "allOf", "oneOf", "anyOf", "if", "then", "else", "discriminator", "propertyNames", "patternProperties", "example", "examples", "default", "const", "enum"} {
			if path == schemaPath && key == "example" {
				continue
			}
			if field(nodes[path], key) != nil {
				return nil, "property has dependent constraints or data requiring coordinated migration"
			}
		}
		i := strings.LastIndex(path, "/")
		if i < 0 {
			break
		}
		path = path[:i]
	}
	ops := []Operation{{Op: "move", From: f.Pointer, Path: parentPath + "/" + EscapeToken(value)}}
	if required := field(schema, "required"); required != nil {
		if required.Kind != yaml.SequenceNode {
			return nil, "required must be an array"
		}
		for i, name := range required.Content {
			if name.Tag == "!!str" && name.Value == old {
				b, _ := json.Marshal(value)
				ops = append(ops, Operation{Op: "replace", Path: schemaPath + "/required/" + strconv.Itoa(i), Value: b})
			}
		}
	}
	if example := field(schema, "example"); example != nil {
		if example.Kind != yaml.MappingNode {
			return nil, "example requires a coordinated migration"
		}
		if field(example, old) != nil {
			if field(example, value) != nil {
				return nil, "normalized example key already exists"
			}
			ops = append(ops, Operation{Op: "move", From: schemaPath + "/example/" + EscapeToken(old), Path: schemaPath + "/example/" + EscapeToken(value)})
		}
	}
	return ops, ""
}

func reviewedHTTPMethod(value string) bool {
	switch value {
	case "get", "put", "post", "patch", "delete", "head", "options", "trace":
		return true
	}
	return false
}
func reviewedDataPath(parts []string) bool {
	for _, p := range parts {
		if strings.HasPrefix(p, "x-") {
			return true
		}
	}
	// Examples, defaults and schema value collections contain user data, not Link Objects.
	for _, p := range parts {
		switch p {
		case "example", "examples", "default", "enum", "const":
			return true
		}
	}
	return false
}

func reviewedStructuralDuplicates(nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	var candidates []string
	for path, node := range nodes {
		if node.Kind != yaml.SequenceNode || !strings.HasSuffix(path, "/required") && !strings.HasSuffix(path, "/enum") {
			continue
		}
		if path == f.Pointer || path == f.Pointer+"/required" || path == f.Pointer+"/enum" || strings.HasPrefix(f.Pointer, path+"/") {
			candidates = append(candidates, path)
		}
	}
	sort.Strings(candidates)
	var ops []Operation
	for _, path := range candidates {
		seq := nodes[path]
		seen := map[string]bool{}
		var duplicates []int
		for i, v := range seq.Content {
			if strings.HasSuffix(path, "/required") && v.Tag != "!!str" {
				return nil, "required array contains invalid names"
			}
			b, err := raw(v)
			if err != nil {
				return nil, "array value cannot be compared"
			}
			var value any
			decoder := json.NewDecoder(bytes.NewReader(b))
			decoder.UseNumber()
			if decoder.Decode(&value) != nil {
				return nil, "array value cannot be compared"
			}
			canonical, _ := json.Marshal(value)
			key := string(canonical)
			if seen[key] {
				if !uncommented(v) {
					return nil, "duplicate carries comments"
				}
				duplicates = append(duplicates, i)
			} else {
				seen[key] = true
			}
		}
		for i := len(duplicates) - 1; i >= 0; i-- {
			ops = append(ops, Operation{Op: "remove", Path: path + "/" + strconv.Itoa(duplicates[i])})
		}
	}
	return ops, "no redundant required or enum entries at the diagnostic location"
}

func reviewedInertConstraint(nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	schema, p := reviewedOwner(nodes, f.Pointer)
	typ := field(schema, "type")
	if typ == nil || typ.Kind != yaml.ScalarNode {
		return nil, "one explicit type is required"
	}
	for _, key := range []string{"$id", "$anchor", "$dynamicAnchor", "$ref", "$dynamicRef", "$recursiveRef"} {
		if field(schema, key) != nil {
			return nil, "scoped schema needs annotation and reference analysis"
		}
	}
	expected := map[string]string{"minimum": "number", "maximum": "number", "exclusiveMinimum": "number", "exclusiveMaximum": "number", "multipleOf": "number", "minLength": "string", "maxLength": "string", "pattern": "string", "minItems": "array", "maxItems": "array", "uniqueItems": "array", "minProperties": "object", "maxProperties": "object", "required": "object"}
	var ops []Operation
	// Only scalar constraints are removed. Subschemas may carry referenced anchors.
	for i := 0; schema != nil && i < len(schema.Content); i += 2 {
		key := schema.Content[i]
		value := schema.Content[i+1]
		kind := expected[key.Value]
		if kind == "" || kind == typ.Value || kind == "number" && typ.Value == "integer" {
			continue
		}
		if value.Kind != yaml.ScalarNode || !uncommented(key) || !uncommented(value) {
			continue
		}
		ops = append(ops, Operation{Op: "remove", Path: p + "/" + EscapeToken(key.Value)})
	}
	return ops, "no inert scalar constraint can be removed"
}

func reviewedPathOrder(root *yaml.Node) ([]Operation, string) {
	paths := field(root, "paths")
	if paths == nil || paths.Kind != yaml.MappingNode || !uncommented(paths) {
		return nil, "path ordering requires an uncommented paths object"
	}
	if len(paths.Content) > 2000 {
		return nil, "path ordering exceeds the 1000-path proposal limit"
	}
	keys := make([]string, 0, len(paths.Content)/2)
	for i := 0; i < len(paths.Content); i += 2 {
		key := paths.Content[i].Value
		if !strings.HasPrefix(key, "/") {
			return nil, "only concrete path entries can be reordered"
		}
		for _, segment := range strings.Split(key, "/") {
			if strings.ContainsAny(segment, "{}") && !(strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") && strings.Count(segment, "{") == 1 && strings.Count(segment, "}") == 1) {
				return nil, "mixed URI template syntax requires a specificity-aware transformation"
			}
		}
		keys = append(keys, key)
	}
	original := strings.Join(keys, "\x00")
	sort.SliceStable(keys, func(i, j int) bool {
		a, b := strings.Split(keys[i], "/"), strings.Split(keys[j], "/")
		for k := 0; k < len(a) && k < len(b); k++ {
			av, bv := strings.HasPrefix(a[k], "{"), strings.HasPrefix(b[k], "{")
			if av != bv {
				return !av
			}
			if !av && a[k] != b[k] {
				return a[k] < b[k]
			}
		}
		return len(a) < len(b)
	})
	if strings.Join(keys, "\x00") == original {
		return nil, "path entries already follow this specificity order"
	}
	var value bytes.Buffer
	value.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			value.WriteByte(',')
		}
		name, _ := json.Marshal(key)
		value.Write(name)
		value.WriteByte(':')
		data, err := raw(field(paths, key))
		if err != nil {
			return nil, err.Error()
		}
		value.Write(data)
	}
	value.WriteByte('}')
	return []Operation{{Op: "replace", Path: "/paths", Value: value.Bytes()}}, ""
}

func reviewedArazzoIdentifier(nodes map[string]*yaml.Node, f Finding) ([]Operation, string) {
	parts, _ := tokens(f.Pointer)
	value := nodes[f.Pointer]
	workflowID := len(parts) == 3 && parts[0] == "workflows" && parts[2] == "workflowId"
	stepID := len(parts) == 5 && parts[0] == "workflows" && parts[2] == "steps" && parts[4] == "stepId"
	if (!workflowID && !stepID) || value == nil || value.Tag != "!!str" || value.Value == "" {
		return nil, "only local workflow and step identifiers have a naming proposal"
	}
	ids := map[string]bool{}
	for path, node := range nodes {
		if strings.Contains(path, "/dependsOn") {
			return nil, "workflow dependencies require a coordinated naming migration"
		}
		if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
			continue
		}
		ps, _ := tokens(path)
		definition := len(ps) == 3 && ps[0] == "workflows" && ps[2] == "workflowId" || len(ps) == 5 && ps[0] == "workflows" && ps[2] == "steps" && ps[4] == "stepId"
		if definition {
			ids[node.Value] = true
			continue
		}
		if strings.Contains(node.Value, "$") || strings.HasSuffix(path, "/workflowId") || strings.HasSuffix(path, "/stepId") {
			return nil, "workflow expressions or references require a coordinated naming migration"
		}
	}
	replacement := reviewedIdentifier(value.Value)
	if replacement == "" {
		replacement = "step"
		if workflowID {
			replacement = "workflow"
		}
	}
	if replacement == value.Value && f.RuleID == "arazzo-advisory" {
		return nil, "identifier already follows the naming convention"
	}
	// The authored array position distinguishes simultaneous duplicate repairs.
	if f.RuleID == "arazzo-duplicate-id" {
		replacement += "_" + parts[len(parts)-2]
	}
	base := replacement
	for i := 2; ids[replacement]; i++ {
		replacement = base + "_" + strconv.Itoa(i)
	}
	b, _ := json.Marshal(replacement)
	return []Operation{{Op: "replace", Path: f.Pointer, Value: b}}, ""
}

func reviewedArazzoSource(root *yaml.Node, nodes map[string]*yaml.Node, f Finding, source string, sources map[string]*yaml.Node) ([]Operation, string) {
	descriptions := field(root, "sourceDescriptions")
	if descriptions == nil || descriptions.Kind != yaml.SequenceNode || len(sources) == 0 {
		return nil, "frozen source descriptions are required"
	}
	resolve := func(description *yaml.Node) *yaml.Node {
		location := field(description, "url")
		if location == nil || location.Tag != "!!str" {
			return nil
		}
		base, err := url.Parse(source)
		if err != nil {
			return nil
		}
		if base.Scheme == "" {
			absolute, err := filepath.Abs(source)
			if err != nil {
				return nil
			}
			base = &url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
		}
		reference, err := url.Parse(location.Value)
		if err != nil {
			return nil
		}
		resolved := base.ResolveReference(reference)
		if resolved.Fragment != "" {
			return nil
		}
		key := resolved.String()
		if resolved.Scheme == "file" {
			key = filepath.FromSlash(resolved.Path)
		}
		doc := sources[key]
		if doc != nil && doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
			doc = doc.Content[0]
		}
		return doc
	}
	if f.RuleID == "arazzo-source-type" {
		owner, p := reviewedOwner(nodes, f.Pointer)
		parts, _ := tokens(p)
		if len(parts) != 2 || parts[0] != "sourceDescriptions" {
			return nil, "source description owner is unavailable"
		}
		doc := resolve(owner)
		if doc == nil {
			return nil, "source document was not captured"
		}
		kind := ""
		for _, entry := range []struct{ field, kind string }{{"openapi", "openapi"}, {"swagger", "openapi"}, {"asyncapi", "asyncapi"}, {"arazzo", "arazzo"}} {
			if field(doc, entry.field) != nil {
				if kind != "" && kind != entry.kind {
					return nil, "source type is ambiguous"
				}
				kind = entry.kind
			}
		}
		if kind == "" {
			return nil, "captured source type is unknown"
		}
		if current := field(owner, "type"); current != nil && current.Value == kind {
			return nil, "source type is already correct"
		}
		return reviewedSet(owner, p, "type", kind), ""
	}
	owner, p := reviewedOwner(nodes, f.Pointer)
	operationPath := field(owner, "operationPath")
	if operationPath == nil || operationPath.Tag != "!!str" || field(owner, "operationId") != nil {
		return nil, "one operationPath without operationId is required"
	}
	expression := operationPath.Value
	if !strings.HasPrefix(expression, "{$sourceDescriptions.") {
		return nil, "operation path must identify one declared source"
	}
	sourceName, fragment, ok := strings.Cut(strings.TrimPrefix(expression, "{$sourceDescriptions."), ".url}#")
	if !ok || !reviewedPortableName.MatchString(sourceName) {
		return nil, "source expression is not a portable exact name"
	}
	var description *yaml.Node
	for _, candidate := range descriptions.Content {
		if name := field(candidate, "name"); name != nil && name.Value == sourceName {
			if description != nil {
				return nil, "source name is ambiguous"
			}
			description = candidate
		}
	}
	if description == nil {
		return nil, "source is not declared"
	}
	document := resolve(description)
	if document == nil {
		return nil, "source document was not captured"
	}
	if field(document, "openapi") == nil && field(document, "swagger") == nil {
		return nil, "operation path proposal currently requires an OpenAPI source"
	}
	pointer, err := url.PathUnescape(fragment)
	if err != nil {
		return nil, "operation pointer is invalid"
	}
	parts, err := tokens(pointer)
	if err != nil || len(parts) != 3 || parts[0] != "paths" || !reviewedHTTPMethod(parts[2]) {
		return nil, "pointer must select a concrete OpenAPI path operation"
	}
	operation, err := at(document, pointer)
	if err != nil || field(operation, "$ref") != nil {
		return nil, "operation requires additional reference resolution"
	}
	id := field(operation, "operationId")
	if id == nil || id.Tag != "!!str" || !reviewedPortableName.MatchString(id.Value) {
		return nil, "target operation needs a portable operationId"
	}
	count := 0
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		if node.Kind == yaml.MappingNode {
			for i := 0; i < len(node.Content); i += 2 {
				if node.Content[i].Value == "operationId" && node.Content[i+1].Value == id.Value {
					count++
				}
			}
		}
		for _, child := range node.Content {
			walk(child)
		}
	}
	walk(document)
	if count != 1 {
		return nil, "target operationId is not unique in the captured source"
	}
	if !uncommented(operationPath) || !mappingKeyUncommented(owner, "operationPath") {
		return nil, "operationPath carries comments"
	}
	ops := reviewedSet(owner, p, "operationId", "$sourceDescriptions."+sourceName+"."+id.Value)
	return append(ops, Operation{Op: "remove", Path: p + "/operationPath"}), ""
}

var reviewedPortableName = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func reviewedDescriptionLabel(parts []string, owner *yaml.Node) string {
	if len(parts) >= 3 && (parts[0] == "paths" || parts[0] == "webhooks") && reviewedHTTPMethod(parts[2]) {
		label := strings.ToUpper(parts[2]) + " " + parts[1]
		if name := field(owner, "name"); name != nil && name.Tag == "!!str" {
			return name.Value + " parameter for " + label
		}
		for _, part := range parts[3:] {
			if part == "schema" {
				return label + " schema"
			}
		}
		for _, part := range parts[3:] {
			if part == "requestBody" {
				return label + " request body"
			}
			if part == "responses" {
				return label + " response"
			}
		}
		return label
	}
	if len(parts) >= 3 && parts[0] == "components" {
		if len(parts) >= 5 && parts[len(parts)-2] == "properties" {
			return parts[len(parts)-1] + " property of " + parts[2]
		}
		kind := map[string]string{"schemas": "schema", "responses": "response", "parameters": "parameter", "headers": "header", "securitySchemes": "security scheme", "requestBodies": "request body", "links": "link", "examples": "example", "callbacks": "callback"}[parts[1]]
		if kind != "" {
			return parts[2] + " " + kind
		}
		return parts[2]
	}
	if len(parts) >= 2 && parts[0] == "definitions" {
		return parts[1] + " schema"
	}
	if len(parts) == 1 && parts[0] == "info" {
		return "API information"
	}
	return parts[len(parts)-1]
}
