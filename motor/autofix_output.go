package motor

import (
	"fmt"
	"github.com/pb33f/go-yaml"
)

// marshalFixedDocument preserves the distinction between no repair and a repair
// whose output cannot be serialized. All document execution paths use it.
func marshalFixedDocument(root *yaml.Node, fixed int) (data []byte, err error) {
	if fixed == 0 {
		return nil, nil
	}
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("serialize repaired document: %v", v)
		}
		if err != nil {
			data = nil
			err = &RuleExecutionError{Kind: RuleErrorAutoFix, Err: err}
		}
	}()
	if root == nil {
		return nil, fmt.Errorf("repaired document is unavailable")
	}
	data, err = yaml.Marshal(root)
	return data, err
}
