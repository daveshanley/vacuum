// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package owasp

import (
	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/doctor/model/high/v3"
	"go.yaml.in/yaml/v4"
)

// LocateSchemaPropertyPaths is a wrapper for the utils version, kept for backwards compatibility
func LocateSchemaPropertyPaths(
	context model.RuleFunctionContext,
	schema *v3.Schema,
	keyNode *yaml.Node,
	valueNode *yaml.Node,
) (primaryPath string, allPaths []string) {
	return vacuumUtils.LocateSchemaPropertyPaths(context, schema, keyNode, valueNode)
}
