package repair

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pb33f/go-yaml"
)

type policyDocument struct {
	Version  int           `yaml:"version"`
	Policies []PolicyEntry `yaml:"policies"`
}

// Policy is constructed by ParsePolicy or NewPolicy. Its snapshots stay private so
// exported plan provenance cannot diverge from the values used for repair.
// Only the caller can supply policy authority; a submitted plan cannot.
type Policy struct {
	entries  []PolicyEntry
	hash     string
	base     string
	catalogs map[string]map[string]map[string]string
}

// PolicyEntry associates a diagnostic with approved metadata or a field map.
// Catalog entries are keyed by source path, then unique operationId.
type PolicyEntry struct {
	ID         string      `yaml:"id"`
	Rule       string      `yaml:"rule"`
	Diagnostic string      `yaml:"diagnostic"`
	Scope      PolicyScope `yaml:"scope"`
	Value      yaml.Node   `yaml:"value"`
	Catalog    string      `yaml:"catalog"`
}

// PolicyScope selects an exact source and JSON Pointer. Globs are not supported.
type PolicyScope struct {
	Source  string `yaml:"source"`
	Pointer string `yaml:"pointer"`
}

// ParsePolicy validates a local policy and its caller-loaded catalog snapshots.
// base is the policy directory; this function never reads files or the network.
func ParsePolicy(data []byte, base string, catalogs map[string][]byte) (*Policy, error) {
	var document policyDocument
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(&document); err != nil {
		return nil, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("policy requires one document")
	}
	return parsePolicyDocument(document, data, base, catalogs)
}

// NewPolicy validates caller-approved values without a serialization round trip.
// The policy owns a copy of each value so later caller edits cannot change it.
func NewPolicy(entries []PolicyEntry, base string) (*Policy, error) {
	owned := append([]PolicyEntry(nil), entries...)
	for i := range owned {
		if err := validateTree(&owned[i].Value, make(map[*yaml.Node]bool)); err != nil {
			return nil, err
		}
		if _, err := treeCost(&owned[i].Value, 0); err != nil {
			return nil, err
		}
		owned[i].Value = *clone(&owned[i].Value)
	}
	data, err := json.Marshal(owned)
	if err != nil {
		return nil, err
	}
	return parsePolicyDocument(policyDocument{Version: 1, Policies: owned}, data, base, nil)
}

func parsePolicyDocument(document policyDocument, data []byte, base string, catalogs map[string][]byte) (*Policy, error) {
	if document.Version != 1 {
		return nil, fmt.Errorf("unsupported policy version %d", document.Version)
	}
	p := Policy{entries: document.Policies}
	p.base = base
	p.catalogs = make(map[string]map[string]map[string]string)
	ids := make(map[string]bool)
	targets := make(map[string]bool)
	var identity bytes.Buffer
	identity.Write(data)
	for i := range p.entries {
		entry := &p.entries[i]
		if entry.ID == "" || ids[entry.ID] {
			return nil, fmt.Errorf("policy ID must be unique and nonempty")
		}
		ids[entry.ID] = true
		if !policyRule(entry.Rule) || entry.Diagnostic != policyField(entry.Rule)+"-missing" {
			return nil, fmt.Errorf("policy requires a supported rule and diagnostic: %s", entry.ID)
		}
		if entry.Scope.Source == "" || strings.ContainsAny(entry.Scope.Source, "*?[") || strings.Contains(entry.Scope.Source, "://") {
			return nil, fmt.Errorf("policy requires an exact local source: %s", entry.ID)
		}
		if _, err := tokens(entry.Scope.Pointer); err != nil {
			return nil, err
		}
		entry.Scope.Source = localPolicyPath(base, entry.Scope.Source)
		key := entry.Rule + "\x00" + entry.Scope.Source + "\x00" + entry.Scope.Pointer
		if targets[key] {
			return nil, fmt.Errorf("overlapping policy target: %s", entry.ID)
		}
		targets[key] = true
		if (entry.Value.Kind == 0) == (entry.Catalog == "") {
			return nil, fmt.Errorf("policy must supply exactly one value or catalog: %s", entry.ID)
		}
		if entry.Value.Kind != 0 {
			if err := validateTree(&entry.Value, make(map[*yaml.Node]bool)); err != nil {
				return nil, fmt.Errorf("invalid policy value %s: %w", entry.ID, err)
			}
			if _, err := raw(&entry.Value); err != nil {
				return nil, fmt.Errorf("invalid policy value %s: %w", entry.ID, err)
			}
			field := policyField(entry.Rule)
			if field == "contact" || field == "license" {
				if entry.Value.Kind != yaml.MappingNode {
					return nil, fmt.Errorf("policy %s requires an object", entry.ID)
				}
				for j := 1; j < len(entry.Value.Content); j += 2 {
					if entry.Value.Content[j].Tag != "!!str" {
						return nil, fmt.Errorf("policy %s requires text fields", entry.ID)
					}
				}
			} else if entry.Value.Kind != yaml.ScalarNode || entry.Value.Tag != "!!str" {
				return nil, fmt.Errorf("policy %s requires text", entry.ID)
			}
		}
		if entry.Catalog != "" {
			if field := policyField(entry.Rule); field == "contact" || field == "license" {
				return nil, fmt.Errorf("policy %s requires an object value, not a text catalog", entry.ID)
			}
			if strings.Contains(entry.Catalog, "://") {
				return nil, fmt.Errorf("remote catalogs are not supported")
			}
			name := localPolicyPath(base, entry.Catalog)
			content, ok := catalogs[name]
			if !ok {
				return nil, fmt.Errorf("catalog snapshot is missing: %s", entry.Catalog)
			}
			if _, err := ParseDocument(content); err != nil {
				return nil, err
			}
			var catalog map[string]map[string]string
			if err := yaml.Unmarshal(content, &catalog); err != nil {
				return nil, err
			}
			normalized := make(map[string]map[string]string)
			for source, entries := range catalog {
				path := localPolicyPath(base, source)
				if normalized[path] != nil {
					return nil, fmt.Errorf("duplicate catalog source")
				}
				normalized[path] = entries
			}
			p.catalogs[entry.Catalog] = normalized
		}
	}
	names := make([]string, 0, len(catalogs))
	for name := range catalogs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintf(&identity, "\x00%d:%s%d:", len(name), name, len(catalogs[name]))
		identity.Write(catalogs[name])
	}
	p.hash = Hash(identity.Bytes())
	return &p, nil
}
func localPolicyPath(base, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	absolute, err := filepath.Abs(path)
	if err == nil {
		return absolute
	}
	return filepath.Clean(path)
}

// ApplyPolicy extends a built-in plan with missing fields supplied by policy.
// Existing conflicting values are retained and the repair is declined.
func ApplyPolicy(plan Plan, data []byte, findings []Finding, policy *Policy) (Plan, error) {
	if policy == nil {
		return plan, nil
	}
	findings = orderedFindings(findings)
	plan.PolicyHash = policy.hash
	parsed, err := ParseDocument(data)
	if err != nil {
		return plan, err
	}
	doc := *parsed
	root := doc.Content[0]
	if err := validateTree(root, make(map[*yaml.Node]bool)); err != nil {
		return plan, nil
	}
	nodes := sourceNodes(root)
	conflicts := &editIndex{}
	for _, op := range plan.Operations {
		conflicts.add(operationFootprint(op, nodes))
		if op.Op == "move" || op.Op == "copy" {
			conflicts.add(sourceFootprint(op, nodes))
		}
	}
	for _, edit := range plan.KeyTags {
		conflicts.add(edit.Parent + "/" + EscapeToken(edit.Key))
	}
	entries := make(map[string][]PolicyEntry)
	for _, entry := range policy.entries {
		key := entry.Rule + "\x00" + entry.Scope.Source + "\x00" + entry.Scope.Pointer
		entries[key] = append(entries[key], entry)
	}
	operationIDs := make(map[string]int)
	for path, node := range nodes {
		parts, _ := tokens(path)
		if !(len(parts) == 3 && (parts[0] == "paths" || parts[0] == "webhooks") && reviewedHTTPMethod(parts[2]) || len(parts) == 4 && parts[0] == "components" && parts[1] == "pathItems" && reviewedHTTPMethod(parts[3])) {
			continue
		}
		if id := field(node, "operationId"); id != nil {
			operationIDs[id.Value]++
		}
	}
	source := localPolicyPath("", plan.Source)
	for _, f := range findings {
		if f.Unavailable != "" {
			continue
		}
		for _, entry := range entries[f.RuleID+"\x00"+source+"\x00"+f.Pointer] {
			if entry.Rule != f.RuleID || entry.Scope.Source != source || entry.Scope.Pointer != f.Pointer || (f.Code != "" && entry.Diagnostic != f.Code) {
				continue
			}
			target := Target{RuleID: f.RuleID, Code: entry.Diagnostic, Pointer: f.Pointer}
			n := nodes[f.Pointer]
			if n == nil {
				continue
			}
			value := &entry.Value
			if entry.Catalog != "" {
				id := field(n, "operationId")
				if id == nil || id.Tag != "!!str" || operationIDs[id.Value] != 1 {
					continue
				}
				text, ok := policy.catalogs[entry.Catalog][source][id.Value]
				if !ok || text == "" {
					continue
				}
				value = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: text}
			}
			fields := value
			if key := policyField(f.RuleID); key != "" {
				fields = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value}}
			}
			if n.Kind != yaml.MappingNode || fields.Kind != yaml.MappingNode {
				continue
			}
			var ops []Operation
			conflict := false
			for i := 0; i < len(fields.Content); i += 2 {
				key, v := fields.Content[i].Value, fields.Content[i+1]
				if existing := field(n, key); existing != nil {
					equal, err := equalNodes(existing, v)
					if err != nil || !equal {
						conflict = true
					}
					continue
				}
				encoded, err := raw(v)
				if err != nil {
					return plan, err
				}
				path := f.Pointer + "/" + EscapeToken(key)
				if conflicts.conflicts(editFootprint(path, nodes)) {
					conflict = true
				}

				ops = append(ops, Operation{Op: "add", Path: path, Value: encoded})
			}
			if conflict || len(ops) == 0 {
				continue
			}
			for _, op := range ops {
				conflicts.add(operationFootprint(op, nodes))
			}
			plan.Operations = append(plan.Operations, ops...)
			plan.Targets = append(plan.Targets, target)
			for _, op := range ops {
				plan.Absent = append(plan.Absent, op.Path)
			}
			remaining := plan.Declined[:0]
			for _, declined := range plan.Declined {
				if declined.Target.RuleID != f.RuleID || declined.Target.Pointer != f.Pointer {
					remaining = append(remaining, declined)
				}
			}
			plan.Declined = remaining
		}
	}
	return plan, nil
}
func policyField(rule string) string {
	switch rule {
	case "info-contact", "asyncapi-info-contact":
		return "contact"
	case "info-license", "asyncapi-info-license":
		return "license"
	case "info-description", "operation-description", "component-description", "oas2-parameter-description", "oas3-parameter-description", "tag-description", "asyncapi-info-description", "asyncapi-3-operation-description", "asyncapi-parameter-description", "asyncapi-3-tag-description", "arazzo-info-description", "arazzo-step-description", "arazzo-workflow-description":
		return "description"
	case "arazzo-info-summary", "arazzo-workflow-summary":
		return "summary"
	case "operation-operationId":
		return "operationId"
	}
	return ""
}

// PolicyCatalogNames returns the local catalog paths needed before ParsePolicy.
// It is only discovery: ParsePolicy must still validate the complete policy.
func PolicyCatalogNames(data []byte, base string) ([]string, error) {
	var p policyDocument
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range p.Policies {
		if entry.Catalog != "" {
			if strings.Contains(entry.Catalog, "://") {
				return nil, fmt.Errorf("remote catalogs are not supported")
			}
			names = append(names, localPolicyPath(base, entry.Catalog))
		}
	}
	sort.Strings(names)
	return names, nil
}

func policyRule(rule string) bool { return policyField(rule) != "" }
