// Package repair plans and applies source-checked document edits without I/O.
// Applying a plan does not verify API semantics; callers must reparse and lint
// the candidate with the original rules and independently approved edit scope.
package repair

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// Version is the supported plan envelope version.
const Version = 1

// Operation is a standard sequential JSON Patch operation. Value is JSON data.
type Operation struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	From  string          `json:"from,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// Target identifies a diagnostic that a plan proposes to repair.
type Target struct {
	RuleID  string `json:"ruleId"`
	Code    string `json:"code"`
	Pointer string `json:"pointer"`
}

// Declined explains why a finding has no proposed edit.
type Declined struct {
	Target Target `json:"target"`
	Reason string `json:"reason"`
}

// KeyTagEdit changes only the YAML tag and style of an existing mapping key.
// Index and key spelling are checked against the original source mapping.
type KeyTagEdit struct {
	Parent string `json:"parent"`
	Index  int    `json:"index"`
	Key    string `json:"key"`
	Tag    string `json:"tag"`
}

// KeyRemoval removes a fully identical repeated YAML mapping entry. It is
// separate from JSON Patch because duplicate members have no unique pointer.
type KeyRemoval struct {
	Parent string          `json:"parent"`
	Index  int             `json:"index"`
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
}

// Plan is an immutable proposal tied to exact source bytes. It grants no write
// authority. Data operations use RFC 6902 ordering; presentation edits apply
// first. Duplicate-key removals must be applied in a separate batch.
type Plan struct {
	// ReviewedProposals keeps each finding's review-only operation group, including
	// groups that conflict in a combined batch. It is private discovery metadata,
	// not a serialized patch or permission to apply unverified operations.
	ReviewedProposals map[string][]Operation `json:"-"`

	Version     int               `json:"version"`
	Source      string            `json:"source"`
	SHA256      string            `json:"sha256"`
	Targets     []Target          `json:"targets,omitempty"`
	Operations  []Operation       `json:"operations,omitempty"`
	Absent      []string          `json:"absent,omitempty"`
	KeyTags     []KeyTagEdit      `json:"keyTags,omitempty"`
	Declined    []Declined        `json:"declined,omitempty"`
	KeyRemovals []KeyRemoval      `json:"keyRemovals,omitempty"`
	Sources     map[string]string `json:"sources,omitempty"`
	PolicyHash  string            `json:"policyHash,omitempty"`
}

// Hash identifies the exact source or policy bytes used to construct a plan.
func Hash(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }

// NewPlan creates an empty proposal for one source snapshot.
func NewPlan(source string, data []byte) Plan {
	return Plan{Version: Version, Source: source, SHA256: Hash(data)}
}

// Apply checks source preconditions and applies the entire plan privately.
// It returns no bytes on error and preserves JSON input as JSON output.
func Apply(source string, data []byte, plan Plan) ([]byte, error) {
	if plan.Version != Version {
		return nil, fmt.Errorf("unsupported repair plan version %d", plan.Version)
	}
	if plan.Source != source || plan.SHA256 != Hash(data) {
		return nil, fmt.Errorf("repair source does not match the plan")
	}
	if len(plan.KeyRemovals) > 0 && (len(plan.KeyTags) > 0 || len(plan.Operations) > 0 || len(plan.Absent) > 0) {
		return nil, fmt.Errorf("duplicate-key removals require a separate repair batch")
	}
	parsed, parseErr := ParseDocument(data)
	if parseErr != nil {
		return nil, parseErr
	}
	doc := *parsed
	if len(doc.Content) != 1 {
		return nil, fmt.Errorf("repair requires one document")
	}
	root := doc.Content[0]
	format := utils.CaptureRepairYAMLFormat(root)
	originalRoot := clone(root)
	if err := validateTreeShape(root, make(map[*yaml.Node]bool), false); err != nil {
		return nil, err
	}
	for _, edit := range plan.KeyRemovals {
		parent, err := at(root, edit.Parent)
		if err != nil {
			return nil, err
		}
		if parent.Kind != yaml.MappingNode || edit.Index <= 0 || edit.Index >= len(parent.Content)/2 {
			return nil, fmt.Errorf("invalid duplicate key removal")
		}
		key, value := parent.Content[2*edit.Index], parent.Content[2*edit.Index+1]
		expected, err := valueNode(edit.Value)
		if err != nil {
			return nil, err
		}
		equal, err := equalNodes(value, expected)
		if err != nil || !equal || key.Value != edit.Key || !uncommented(key) || !uncommented(value) {
			return nil, fmt.Errorf("duplicate key precondition failed")
		}
		duplicate := false
		for i := 0; i < 2*edit.Index; i += 2 {
			if parent.Content[i].Value == key.Value && parent.Content[i].Tag == key.Tag {
				same, err := equalNodes(parent.Content[i+1], value)
				if err != nil {
					return nil, err
				}
				if same {
					duplicate = true
					break
				}
			}
		}
		if !duplicate {
			return nil, fmt.Errorf("mapping entry is not an identical duplicate")
		}
		parent.Content = append(parent.Content[:2*edit.Index], parent.Content[2*edit.Index+2:]...)
	}
	if err := validateTree(root, make(map[*yaml.Node]bool)); err != nil {
		return nil, err
	}
	for _, path := range plan.Absent {
		parent, last, err := parentAt(root, path)
		if err != nil {
			return nil, err
		}
		if parent.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("absence precondition requires an object: %s", path)
		}
		if _, ok := mapIndex(parent, last); ok {
			return nil, fmt.Errorf("expected absent member: %s", path)
		}
	}
	for _, edit := range plan.KeyTags {
		parent, err := at(root, edit.Parent)
		if err != nil {
			return nil, err
		}
		if parent.Kind != yaml.MappingNode || edit.Index < 0 || edit.Index >= len(parent.Content)/2 {
			return nil, fmt.Errorf("invalid key edit at %s", edit.Parent)
		}
		key := parent.Content[2*edit.Index]
		if key.Value != edit.Key || key.Tag != edit.Tag || key.Tag != "!!int" {
			return nil, fmt.Errorf("key edit precondition failed at %s", edit.Parent)
		}
		key.Tag = "!!str"
		key.Style = yaml.DoubleQuotedStyle
	}
	if len(plan.Operations) > 10000 {
		return nil, fmt.Errorf("repair exceeds 10000 operations")
	}
	cost, err := treeCost(root, 0)
	if err != nil {
		return nil, err
	}
	for _, op := range plan.Operations {
		if op.Op == "copy" {
			n, err := at(root, op.From)
			if err != nil {
				return nil, err
			}
			added, err := treeCost(n, 0)
			if err != nil {
				return nil, err
			}
			cost += added
		} else if op.Op == "add" || op.Op == "replace" {
			if len(op.Value) > 64<<20 {
				return nil, fmt.Errorf("repair value exceeds budget")
			}
			n, err := valueNode(op.Value)
			if err != nil {
				return nil, err
			}
			added, err := treeCost(n, 0)
			if err != nil {
				return nil, err
			}
			cost += added
		}
		if cost > 256<<20 {
			return nil, fmt.Errorf("repair exceeds memory budget")
		}

		var err error
		root, err = applyOperation(root, op)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", op.Op, op.Path, err)
		}
	}
	doc.Content[0] = root
	if len(plan.Operations) == 0 && len(plan.KeyTags) == 0 && len(plan.KeyRemovals) == 0 {
		return append([]byte(nil), data...), nil
	}
	return renderSource(originalRoot, root, data, json.Valid(data), format)
}

func validateTree(n *yaml.Node, seen map[*yaml.Node]bool) error {
	return validateTreeShape(n, seen, true)
}
func validateTreeShape(n *yaml.Node, seen map[*yaml.Node]bool, unique bool) error {
	if n == nil || seen[n] {
		return fmt.Errorf("repair requires an unaliased document tree")
	}
	seen[n] = true
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return fmt.Errorf("repair of anchored or aliased documents requires review")
	}
	if n.Kind == yaml.MappingNode {
		if len(n.Content)%2 != 0 {
			return fmt.Errorf("invalid mapping")
		}
		keys := make(map[string]bool)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || (key.Tag != "!!str" && key.Tag != "!!int") {
				return fmt.Errorf("unsupported mapping key")
			}
			if unique && keys[key.Value] {
				return fmt.Errorf("duplicate mapping key %q requires a source edit", key.Value)
			}
			keys[key.Value] = true
		}
	}
	for _, child := range n.Content {
		if err := validateTreeShape(child, seen, unique); err != nil {
			return err
		}
	}
	return nil
}
