package core

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
)

func fieldLookupOptions(context model.RuleFunctionContext, recursiveFirstSegment bool) vacuumUtils.FieldPathOptions {
	return vacuumUtils.FieldPathOptions{
		RecursiveFirstSegment:        recursiveFirstSegment,
		ResolveSingleItemCombinators: context.Rule != nil && context.Rule.Resolved,
	}
}
