package motor

import "github.com/daveshanley/vacuum/repair"

// The provider identifies a single value edit. Compound transforms must retain
// their linked edits and are never reduced to an independently editable field.
func editableProposalOperation(plan repair.Plan, rule, pointer, metadataField string) (repair.Operation, bool) {
	if group := plan.ReviewedProposals[rule+"\x00"+pointer]; len(group) > 0 {
		if len(group) == 1 && (group[0].Op == "add" || group[0].Op == "replace") {
			return group[0], true
		}
		return repair.Operation{}, false
	}
	if metadataField == "" {
		return repair.Operation{}, false
	}
	path := pointer + "/" + repair.EscapeToken(metadataField)
	for _, op := range plan.Operations {
		if op.Path == path && (op.Op == "add" || op.Op == "replace") {
			return op, true
		}
	}
	return repair.Operation{}, false
}
