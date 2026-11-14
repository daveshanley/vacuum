// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package openapi

import (
	"fmt"

	"regexp"

	"strings"

	"github.com/daveshanley/vacuum/model"
	whatwg "github.com/nlnwa/whatwg-url/url"
	"github.com/pb33f/go-yaml"
)

var packagedServerVariable = regexp.MustCompile(`\{(.+?)\}`)

func validPackagedServer(server packagedOperand) bool {
	if !server.field("url").isText() {
		return false
	}
	variables := server.field("variables")
	if variables.node == nil {
		return true
	}
	if !variables.is(yaml.MappingNode) {
		return false
	}
	for _, variable := range variables.fields() {
		if !variable.is(yaml.MappingNode) {
			return false
		}
		for key, field := range variable.fields() {
			switch key {
			case "enum":
				if !field.is(yaml.SequenceNode) {
					return false
				}
				for item := range field.items() {
					if !item.isText() {
						return false
					}
				}
			case "default", "description", "examples":
				if !field.isText() {
					return false
				}
			default:
				if !strings.HasPrefix(key, "x-") {
					return false
				}
			}
		}
	}
	return true
}

type urlVariable struct {
	placeholder string
	values      []string
}

func checkPackagedServerVariables(server packagedOperand, ctx model.RuleFunctionContext, results *packagedDiagnostics) {
	if !validPackagedServer(server) {
		results.add(server, "Invalid Server Object")
		return
	}
	template := server.field("url").text()
	variables := server.field("variables")
	found := make(map[string]bool)
	var missing []string
	for _, match := range packagedServerVariable.FindAllStringSubmatch(template, -1) {
		name := match[1]
		found[name] = true
		if variables.field(name).node == nil {
			missing = append(missing, name)
		}
	}
	var substitutions []urlVariable
	options := ctx.GetOptionsStringMap()
	for name, v := range variables.fields() {
		if !found[name] {
			results.add(v, fmt.Sprintf("Server's \"variables\" object has unused defined %q url variable.", name))
			continue
		}
		values := urlVariable{placeholder: "{" + name + "}"}
		def, enum := v.field("default"), v.field("enum")
		if enum.node != nil {
			for item := range enum.items() {
				values.values = append(values.values, encodeServerVariable(item.text()))
			}
			if def.node != nil && !enum.containsText(def.text()) {
				results.add(def, fmt.Sprintf("Server Variable %q has a default not listed in the enum.", name))
			}
		} else if def.node != nil {
			values.values = []string{encodeServerVariable(def.text())}
		}
		if def.node == nil && options["requireDefault"] == "true" {
			results.add(v, fmt.Sprintf("Server Variable %q has a missing default.", name))
		}
		substitutions = append(substitutions, values)
	}
	at := server.fieldOrSelf("variables")
	if len(missing) > 0 {
		results.add(at, "Not all server's variables are described with \"variables\" object. Missed: "+strings.Join(missing, ", ")+".")
	}
	if options["checkSubstitutions"] != "true" || len(substitutions) == 0 {
		return
	}
	for _, v := range substitutions {
		if len(v.values) == 0 {
			return
		}
	}
	counts := make([]int, len(substitutions))
	var invalid []string
	for {
		if ctx.ExecutionContext != nil && ctx.ExecutionContext.Err() != nil {
			return
		}
		value := template
		for i, v := range substitutions {
			value = strings.ReplaceAll(value, v.placeholder, v.values[counts[i]])
		}
		if !validPackagedServerURL(value) {
			invalid = append(invalid, value)
			if len(invalid) == 5 {
				results.add(at, "At least 5 substitutions of server variables resulted in invalid URLs: "+strings.Join(invalid, ", ")+" and more")
				return
			}
		}
		i := len(counts) - 1
		for ; i >= 0; i-- {
			counts[i]++
			if counts[i] < len(substitutions[i].values) {
				break
			}
			counts[i] = 0
		}
		if i < 0 {
			break
		}
	}
	if len(invalid) > 0 {
		results.add(at, "A few substitutions of server variables resulted in invalid URLs: "+strings.Join(invalid, ", "))
	}
}

// OpenLint calls encodeURI on each substitution before checking the absolute URL.
func encodeServerVariable(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune(";,/?:@&=+$-_.!~*'()#", rune(c)) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

func validPackagedServerURL(raw string) bool {
	_, err := whatwg.Parse(raw)
	return err == nil
}
