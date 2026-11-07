package motor

import (
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/repair"
	"github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
)

// Filter against the authored document. An index in an ignore file identifies
// the original node, not whichever node occupies that index after a repair.
func filterRepairCandidateIgnores(before, after *RuleSetExecutionResult, changes *repair.Correspondence) {
	original := before.RuleSetExecution
	matcher := utils.NewIgnoreMatcher(original.RepairIgnores, utils.IgnoreMatcherOptions{RootNode: original.CanonicalDocument, SpecBytes: original.Spec, LookupTimeout: original.NodeLookupTimeout})
	filtered := make([]model.RuleFunctionResult, 0, len(after.Results))
	for _, result := range after.Results {
		mapped := result
		mapped.Path = originalRepairPath(result.Path, original.CanonicalDocument, after.RuleSetExecution.CanonicalDocument, changes)
		mapped.Paths = make([]string, len(result.Paths))
		for i, path := range result.Paths {
			mapped.Paths[i] = originalRepairPath(path, original.CanonicalDocument, after.RuleSetExecution.CanonicalDocument, changes)
		}
		if !matcher.Matches(&mapped) {
			filtered = append(filtered, result)
		}
	}
	after.Results = filtered
	after.RepairIgnoresApplied = true
}

func originalRepairPath(path string, root, candidateRoot *yaml.Node, changes *repair.Correspondence) string {
	segments, err := utils.ParseFieldPath(strings.TrimPrefix(strings.TrimPrefix(path, "$"), "."))
	if err != nil {
		return ""
	}
	pointer := ""
	for _, segment := range segments {
		token := segment.Key
		if segment.Type == utils.SegmentArrayIndex {
			token = strconv.Itoa(segment.Index)
		}
		pointer += "/" + repair.EscapeToken(token)
	}
	// Findings for missing fields can name an absent child. Translate the nearest
	// surviving ancestor, then retain that absent suffix.
	suffix := ""
	for {
		if origin, ok := changes.OriginalPointer(pointer); ok {
			pointer = origin + suffix
			break
		}
		if fixNodeAt(candidateRoot, pointer) != nil {
			return ""
		}
		at := strings.LastIndexByte(pointer, '/')
		if at < 0 {
			return ""
		}
		suffix = pointer[at:] + suffix
		pointer = pointer[:at]
	}
	node := root
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	result := "$"
	for _, token := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		if pointer == "" {
			break
		}
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		if node != nil && node.Kind == yaml.SequenceNode {
			index, err := strconv.Atoi(token)
			if err != nil {
				return ""
			}
			result = utils.AppendResultPathIndex(result, index)
			if index >= 0 && index < len(node.Content) {
				node = node.Content[index]
			} else {
				node = nil
			}
		} else {
			result = utils.AppendResultPathSegment(result, token)
			node = fixNodeAt(node, "/"+repair.EscapeToken(token))
		}
	}
	return result
}
