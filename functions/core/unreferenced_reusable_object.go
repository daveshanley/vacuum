package core

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/daveshanley/vacuum/model"
	vacuumUtils "github.com/daveshanley/vacuum/utils"
	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/index"
)

// UnreferencedReusableObject reports members of a reusable-object container
// that have no reference in the indexed document graph.
type UnreferencedReusableObject struct{}

// GetSchema describes the required JSON Pointer to the reusable container.
func (UnreferencedReusableObject) GetSchema() model.RuleFunctionSchema {
	return model.RuleFunctionSchema{Name: "unreferencedReusableObject", Required: []string{"reusableObjectsLocation"}, Properties: []model.RuleFunctionProperty{{Name: "reusableObjectsLocation", Description: "Local JSON Pointer to the reusable-object container."}}}
}

// GetCategory returns the core function category.
func (UnreferencedReusableObject) GetCategory() string { return model.FunctionCategoryCore }

// ValidateOptions requires a local JSON Pointer, including escaped property names.
func (UnreferencedReusableObject) ValidateOptions(options any) error {
	opts, ok := options.(map[string]any)
	if !ok {
		return fmt.Errorf("reusableObjectsLocation must be a local JSON Pointer")
	}
	location, ok := opts["reusableObjectsLocation"].(string)
	if !ok || !strings.HasPrefix(location, "#/") {
		return fmt.Errorf("reusableObjectsLocation must be a local JSON Pointer")
	}
	decoded, err := url.PathUnescape(location)
	if err != nil {
		return fmt.Errorf("invalid reusableObjectsLocation: %w", err)
	}
	for i := 0; i < len(decoded); i++ {
		if decoded[i] == '~' {
			if i+1 == len(decoded) || decoded[i+1] != '0' && decoded[i+1] != '1' {
				return fmt.Errorf("invalid JSON Pointer escape in reusableObjectsLocation")
			}
			i++
		}
	}
	return nil
}

// RunRule uses the existing reference index. It does not resolve or fetch files.
func (u UnreferencedReusableObject) RunRule(nodes []*yaml.Node, context model.RuleFunctionContext) []model.RuleFunctionResult {
	if len(nodes) == 0 {
		return nil
	}
	if err := u.ValidateOptions(context.Options); err != nil {
		return reusableObjectError(context, err)
	}
	if context.Index == nil {
		return reusableObjectError(context, fmt.Errorf("unreferencedReusableObject requires a document index"))
	}
	location := context.Options.(map[string]any)["reusableObjectsLocation"].(string)
	location, _ = url.PathUnescape(location)
	path := "$"
	for _, segment := range strings.Split(strings.TrimPrefix(location, "#/"), "/") {
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		path = model.AppendResultPathSegment(path, segment)
	}
	source := context.Index.GetSpecAbsolutePath()
	references := make(map[string]bool)
	collect := func(idx *index.SpecIndex) {
		if idx == nil {
			return
		}
		for _, ref := range idx.GetRawReferencesSequenced() {
			target := ref.FullDefinition
			if target == "" {
				target = ref.Definition
			}
			file, fragment, ok := strings.Cut(target, "#")
			if !ok {
				continue
			}
			if file == "" {
				file = idx.GetSpecAbsolutePath()
			}
			if file != source {
				continue
			}
			fragment, err := url.PathUnescape(fragment)
			if err != nil {
				continue
			}
			references["#"+fragment] = true
		}
	}
	collect(context.Index)
	if rolodex := context.Index.GetRolodex(); rolodex != nil {
		for _, idx := range rolodex.GetIndexes() {
			if idx != context.Index {
				collect(idx)
			}
		}
	}
	var results []model.RuleFunctionResult
	for _, node := range nodes {
		if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
			node = node.Content[0]
		}
		if node.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			escaped := strings.ReplaceAll(strings.ReplaceAll(key.Value, "~", "~0"), "/", "~1")
			if references[location+"/"+escaped] {
				continue
			}
			results = append(results, model.RuleFunctionResult{Rule: context.Rule, Path: model.AppendResultPathSegment(path, key.Value), StartNode: key, EndNode: vacuumUtils.BuildEndNode(key), Message: "Potential orphaned reusable object has been detected"})
		}
	}
	return results
}

func reusableObjectError(context model.RuleFunctionContext, err error) []model.RuleFunctionResult {
	if context.ReportError != nil {
		context.ReportError(err)
	}
	return []model.RuleFunctionResult{{Rule: context.Rule, Path: givenPathValue(context.Given), Message: err.Error()}}
}
