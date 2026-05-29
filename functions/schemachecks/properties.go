// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package schemachecks

import (
	highBase "github.com/pb33f/libopenapi/datamodel/high/base"
	lowBase "github.com/pb33f/libopenapi/datamodel/low/base"
)

func checkSchemaPropertyRecursive(schema *highBase.Schema, propertyName string, visited map[*lowBase.Schema]struct{}) bool {
	if schema == nil {
		return false
	}

	if low := schema.GoLow(); low != nil {
		if _, seen := visited[low]; seen {
			return false
		}
		visited[low] = struct{}{}
	}

	if schema.Properties != nil && schema.Properties.GetOrZero(propertyName) != nil {
		return true
	}

	if checkSchemaProxiesForProperty(schema.AnyOf, propertyName, visited) {
		return true
	}
	if checkSchemaProxiesForProperty(schema.OneOf, propertyName, visited) {
		return true
	}
	if checkSchemaProxiesForProperty(schema.AllOf, propertyName, visited) {
		return true
	}

	return false
}

func checkSchemaProxiesForProperty(
	proxies []*highBase.SchemaProxy,
	propertyName string,
	visited map[*lowBase.Schema]struct{},
) bool {
	for _, proxy := range proxies {
		if proxy == nil {
			continue
		}

		subSchema := proxy.Schema()
		if subSchema == nil {
			continue
		}

		if checkSchemaPropertyRecursive(subSchema, propertyName, visited) {
			return true
		}
	}

	return false
}
