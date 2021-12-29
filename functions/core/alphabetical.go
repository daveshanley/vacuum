// Copyright 2020-2021 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package core

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"gopkg.in/yaml.v3"
	"sort"
	"strconv"
	"strings"
)

// Alphabetical is a rule that will check an array or object to determine if the values are in order.
// if the path is to an object, then the value function option 'keyedBy' must be used, to know how to sort the
// data.
type Alphabetical struct{}

// GetSchema returns a model.RuleFunctionSchema defining the schema of the 'Alphabetical' rule.
func (a Alphabetical) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{
		Name: "alphabetical",
		Properties: []model.RuleFunctionProperty{
			{
				Name:        "keyedBy",
				Description: "this is the key of an object you want to use to sort objects",
			},
		},
		ErrorMessage: "'alphabetical' function has invalid options supplied. To sort objects use 'keyedBy'" +
			"and decide which property on the array of objects you want to use.",
	}
}

// RunRule will execute the 'Alphabetical' rule, based on supplied context and a supplied []*yaml.Node slice.
func (a Alphabetical) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	if len(nodes) <= 0 {
		return nil
	}

	var keyedBy string

	// check supplied type
	props := utils.ConvertInterfaceIntoStringMap(context.Options)
	if props["keyedBy"] != "" {
		keyedBy = props["keyedBy"]
	}

	for _, node := range nodes {

		if utils.IsNodeMap(node) {

			if keyedBy == "" {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("'%s' is a map/object. %s",
						node.Value, a.GetSchema().ErrorMessage),
				})
				continue
			}

			resultsFromKey := a.processMap(node, keyedBy)
			results = compareStringArray(resultsFromKey)
			continue
		}

		if utils.IsNodeArray(node) {
			if a.isValidArray(node) {
				if a.isValidStringArray(node) {
					rs := a.checkStringArrayIsSorted(node)
					results = append(results, rs...)
				}

				if a.isValidNumberArray(node) {
					rs := a.checkNumberArrayIsSorted(node)
					results = append(results, rs...)
				}

				if a.isValidMapArray(node) {
					resultsFromKey := a.processMap(node, keyedBy)
					results = compareStringArray(resultsFromKey)
				}

			}
			continue
		}

		// TODO: handle single value code and object code,

	}

	return results
}

func (a Alphabetical) processMap(node *yaml.Node, keyedBy string) []string {
	var resultsFromKey []string
	for x, v := range node.Content {
		// run odd numbers for values.
		if x == 0 || x%2 != 0 {
			if v.Tag == "!!map" {

				for y, kv := range v.Content {

					// check keys for keyedBy match
					if y%2 == 0 && keyedBy == kv.Value && y+1 < len(v.Content) {
						resultsFromKey = append(resultsFromKey, v.Content[y+1].Value)
					}
				}
			}
		}
	}
	return resultsFromKey
}

func (a Alphabetical) isValidArray(arr *yaml.Node) bool {
	for _, n := range arr.Content {
		switch n.Tag {
		case "!!bool":
			return false
		}
	}
	return true
}

func (a Alphabetical) isValidStringArray(arr *yaml.Node) bool {
	return arr.Content[0].Tag == "!!str"
}

func (a Alphabetical) isValidNumberArray(arr *yaml.Node) bool {
	return arr.Content[0].Tag == "!!int" || arr.Content[0].Tag == "!!float"
}

func (a Alphabetical) isValidMapArray(arr *yaml.Node) bool {
	return arr.Content[0].Tag == "!!map"
}

func (a Alphabetical) checkStringArrayIsSorted(arr *yaml.Node) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	var strArr []string
	for _, n := range arr.Content {
		if n.Tag == "!!str" {
			strArr = append(strArr, n.Value)
		}
	}
	if sort.StringsAreSorted(strArr) {
		return nil
	} else {
		results = compareStringArray(strArr)
	}

	return results
}

func compareStringArray(strArr []string) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for x := 0; x < len(strArr); x++ {
		if x+1 < len(strArr) {
			s := strings.Compare(strArr[x], strArr[x+1])
			if s > 0 {
				results = append(results, model.RuleFunctionResult{
					Message: fmt.Sprintf("'%s' must be placed before '%s' (alphabetical)",
						strArr[x+1], strArr[x]),
				})
			}
		}
	}
	return results
}

func (a Alphabetical) checkNumberArrayIsSorted(arr *yaml.Node) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	var intArray []int
	var floatArray []float64

	for _, n := range arr.Content {
		if n.Tag == "!!int" {
			intVal, _ := strconv.Atoi(n.Value)
			intArray = append(intArray, intVal)
		}
		if n.Tag == "!!float" {
			floatVal, _ := strconv.ParseFloat(n.Value, 64)
			floatArray = append(floatArray, floatVal)
		}
	}

	errmsg := "'%v' is less than '%v', they need to be swapped (numerical ordering)"

	if len(floatArray) > 0 {
		if !sort.Float64sAreSorted(floatArray) {
			results = a.evaluateFloatArray(floatArray, errmsg)
		}
	}

	if len(intArray) > 0 {
		if !sort.IntsAreSorted(intArray) {
			results = append(results, a.evaluateIntArray(intArray, errmsg)...)
		}
	}

	return results
}

func (a Alphabetical) evaluateIntArray(intArray []int, errmsg string) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for x, n := range intArray {
		if x+1 < len(intArray) && n > intArray[x+1] {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf(errmsg, intArray[x+1], intArray[x]),
			})
		}
	}
	return results
}

func (a Alphabetical) evaluateFloatArray(floatArray []float64, errmsg string) []model.RuleFunctionResult {
	var results []model.RuleFunctionResult
	for x, n := range floatArray {
		if x+1 < len(floatArray) && n > floatArray[x+1] {
			results = append(results, model.RuleFunctionResult{
				Message: fmt.Sprintf(errmsg, floatArray[x+1], floatArray[x]),
			})
		}
	}
	return results
}
