package transform

import (
	"fmt"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

// ComponentGraph is a read-only reachability graph for reusable components.
type ComponentGraph struct {
	components  map[ComponentID]*yaml.Node
	adjacency   map[ComponentID]map[ComponentID]struct{}
	roots       map[ComponentID]struct{}
	locations   map[ComponentID][]string
	sections    map[string]bool
	yamlAnchors map[*yaml.Node]ComponentID
}

// BuildComponentGraph validates local component references and builds a graph
// whose roots are references made by the retained, non-component document.
func BuildComponentGraph(root *yaml.Node, specVersion string) (*ComponentGraph, error) {
	root, err := requireMappingRoot(root)
	if err != nil {
		return nil, err
	}
	if err := ValidateBundled(root); err != nil {
		return nil, err
	}
	g := &ComponentGraph{
		components:  make(map[ComponentID]*yaml.Node),
		adjacency:   make(map[ComponentID]map[ComponentID]struct{}),
		roots:       make(map[ComponentID]struct{}),
		locations:   make(map[ComponentID][]string),
		sections:    openAPIComponentSectionSet(specVersion),
		yamlAnchors: make(map[*yaml.Node]ComponentID),
	}
	swagger := strings.HasPrefix(specVersion, "2.")
	g.registerComponents(root, swagger)
	g.indexYAMLAnchors()
	anchors := g.buildAnchorIndex()

	if err := g.walkNonComponents(root, nil, swagger, anchors); err != nil {
		return nil, err
	}
	for id, node := range g.components {
		if err := g.walkOwned(node, g.locations[id], &id, swagger, anchors); err != nil {
			return nil, err
		}
	}
	return g, nil
}

func (g *ComponentGraph) indexYAMLAnchors() {
	for id, component := range g.components {
		walkYAMLNodes(component, func(node *yaml.Node) {
			if node.Anchor != "" {
				g.yamlAnchors[node] = id
			}
		})
	}
}

// Unreachable returns registered components not reachable from document roots.
func (g *ComponentGraph) Unreachable() []ComponentID {
	reachable := make(map[ComponentID]struct{}, len(g.components))
	stack := make([]ComponentID, 0, len(g.roots))
	for id := range g.roots {
		stack = append(stack, id)
	}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := reachable[id]; seen {
			continue
		}
		reachable[id] = struct{}{}
		for target := range g.adjacency[id] {
			stack = append(stack, target)
		}
	}
	var result []ComponentID
	for id := range g.components {
		if _, ok := reachable[id]; !ok {
			result = append(result, id)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Section == result[j].Section {
			return result[i].Name < result[j].Name
		}
		return result[i].Section < result[j].Section
	})
	return result
}

func (g *ComponentGraph) registerComponents(root *yaml.Node, swagger bool) {
	if swagger {
		for _, section := range []string{"definitions", "parameters", "responses", "securityDefinitions"} {
			g.registerSection(mapValue(root, section), section, []string{section})
		}
		return
	}
	components := mapValue(root, "components")
	if components == nil || components.Kind != yaml.MappingNode {
		return
	}
	for section := range g.sections {
		g.registerSection(mapValue(components, section), section, []string{"components", section})
	}
}

var baseOpenAPIComponentSections = []string{
	"schemas", "responses", "parameters", "examples", "requestBodies", "headers", "securitySchemes", "links", "callbacks",
}

func (g *ComponentGraph) registerSection(sectionNode *yaml.Node, section string, path []string) {
	if sectionNode == nil || sectionNode.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(sectionNode.Content); i += 2 {
		id := ComponentID{Section: section, Name: sectionNode.Content[i].Value}
		g.components[id] = sectionNode.Content[i+1]
		g.locations[id] = appendPath(path, id.Name)
	}
}

func (g *ComponentGraph) buildAnchorIndex() map[string][]ComponentID {
	anchorSets := make(map[string]map[ComponentID]struct{})
	for id, node := range g.components {
		if id.Section != "schemas" && id.Section != "definitions" {
			continue
		}
		walkYAML(node, func(mapping *yaml.Node) {
			for _, key := range []string{"$anchor", "$dynamicAnchor"} {
				if value := mapValue(mapping, key); value != nil && value.Kind == yaml.ScalarNode && value.Value != "" {
					if anchorSets[value.Value] == nil {
						anchorSets[value.Value] = make(map[ComponentID]struct{})
					}
					anchorSets[value.Value][id] = struct{}{}
				}
			}
		})
	}
	anchors := make(map[string][]ComponentID, len(anchorSets))
	for name, ids := range anchorSets {
		for id := range ids {
			anchors[name] = append(anchors[name], id)
		}
	}
	return anchors
}

func (g *ComponentGraph) walkNonComponents(root *yaml.Node, path []string, swagger bool, anchors map[string][]ComponentID) error {
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		if swagger && isSwaggerComponentSection(key) {
			continue
		}
		if !swagger && key == "components" && value.Kind == yaml.MappingNode {
			for j := 0; j+1 < len(value.Content); j += 2 {
				section := value.Content[j].Value
				if g.sections[section] {
					continue
				}
				if err := g.walkOwned(value.Content[j+1], []string{"components", section}, nil, swagger, anchors); err != nil {
					return err
				}
			}
			continue
		}
		if err := g.walkOwned(value, appendPath(path, key), nil, swagger, anchors); err != nil {
			return err
		}
	}
	return g.addSecurityEdges(root, nil, swagger, nil)
}

func (g *ComponentGraph) walkOwned(node *yaml.Node, path []string, owner *ComponentID, swagger bool, anchors map[string][]ComponentID) error {
	if isWithinArbitraryExample(path) {
		return nil
	}
	inExtension := isWithinExtension(path)
	switch node.Kind {
	case yaml.MappingNode:
		if isSecurityRequirementOwner(path, owner) {
			if err := g.addSecurityEdges(node, owner, swagger, path); err != nil {
				return err
			}
		}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i].Value, node.Content[i+1]
			path = append(path, key)
			next := path
			if key == "operationRef" && isLinkObjectPath(path[:len(path)-1]) && value.Kind == yaml.ScalarNode && strings.HasPrefix(value.Value, "#") {
				if _, err := localPointerTokens(value.Value); err != nil {
					return fmt.Errorf("%s at %s", err, jsonPath(next))
				}
			}
			if documentReferenceKeys[key] && key != "operationRef" && value.Kind == yaml.ScalarNode &&
				!(inExtension && isExternalReference(value.Value)) {
				target, err := g.resolveReference(value.Value, key, owner, swagger, anchors)
				if err != nil {
					return fmt.Errorf("%s at %s", err, jsonPath(next))
				}
				if target != nil {
					if err := g.addEdge(owner, *target, value.Value, next); err != nil {
						return err
					}
				}
			}
			if key == "discriminator" && !inExtension {
				if err := g.addDiscriminatorEdges(value, next, owner, swagger); err != nil {
					return err
				}
			}
			if err := g.walkOwned(value, next, owner, swagger, anchors); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			path = append(path, fmt.Sprintf("%d", i))
			if err := g.walkOwned(child, path, owner, swagger, anchors); err != nil {
				return err
			}
			path = path[:len(path)-1]
		}
	case yaml.AliasNode:
		if target, ok := g.yamlAnchors[node.Alias]; ok {
			if owner == nil {
				g.roots[target] = struct{}{}
			} else {
				if g.adjacency[*owner] == nil {
					g.adjacency[*owner] = make(map[ComponentID]struct{})
				}
				g.adjacency[*owner][target] = struct{}{}
			}
		}
	}
	return nil
}

func (g *ComponentGraph) resolveReference(value, keyword string, owner *ComponentID, swagger bool, anchors map[string][]ComponentID) (*ComponentID, error) {
	tokens, err := localPointerTokens(value)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		if keyword == "$dynamicRef" || keyword == "$recursiveRef" {
			if owner != nil && (owner.Section == "schemas" || owner.Section == "definitions") {
				id := *owner
				return &id, nil
			}
			return nil, fmt.Errorf("cannot statically resolve local recursive reference %q", value)
		}
		return nil, nil
	}
	if len(tokens) == 1 && !strings.HasPrefix(strings.TrimPrefix(value, "#"), "/") {
		if keyword != "$dynamicRef" && keyword != "$recursiveRef" {
			return nil, nil
		}
		ids := anchors[tokens[0]]
		if len(ids) != 1 {
			return nil, fmt.Errorf("cannot statically resolve local anchor reference %q", value)
		}
		id := ids[0]
		return &id, nil
	}
	var id ComponentID
	switch {
	case swagger && len(tokens) >= 2 && isSwaggerComponentSection(tokens[0]):
		id = ComponentID{Section: tokens[0], Name: tokens[1]}
	case !swagger && len(tokens) >= 3 && tokens[0] == "components" && g.sections[tokens[1]]:
		id = ComponentID{Section: tokens[1], Name: tokens[2]}
	case !swagger && len(tokens) > 0 && tokens[0] == "components":
		return nil, fmt.Errorf("invalid local component reference %q", value)
	default:
		return nil, nil
	}
	return &id, nil
}

func (g *ComponentGraph) addEdge(owner *ComponentID, target ComponentID, raw string, path []string) error {
	if _, exists := g.components[target]; !exists {
		return fmt.Errorf("local component reference at %s targets missing component %s/%s: %s", jsonPath(path), target.Section, target.Name, raw)
	}
	if owner == nil {
		g.roots[target] = struct{}{}
		return nil
	}
	if g.adjacency[*owner] == nil {
		g.adjacency[*owner] = make(map[ComponentID]struct{})
	}
	g.adjacency[*owner][target] = struct{}{}
	return nil
}

func (g *ComponentGraph) addDiscriminatorEdges(node *yaml.Node, path []string, owner *ComponentID, swagger bool) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	add := func(value *yaml.Node, valuePath []string) error {
		if value.Kind != yaml.ScalarNode || value.Value == "" {
			return nil
		}
		var id ComponentID
		if strings.HasPrefix(value.Value, "#") {
			tokens, err := localPointerTokens(value.Value)
			if err != nil {
				return fmt.Errorf("%s at %s", err, jsonPath(valuePath))
			}
			if swagger && len(tokens) == 2 && tokens[0] == "definitions" {
				id = ComponentID{Section: "definitions", Name: tokens[1]}
			} else if !swagger && len(tokens) == 3 && tokens[0] == "components" && tokens[1] == "schemas" {
				id = ComponentID{Section: "schemas", Name: tokens[2]}
			} else {
				return fmt.Errorf("invalid local discriminator reference at %s: %s", jsonPath(valuePath), value.Value)
			}
		} else {
			section := "schemas"
			if swagger {
				section = "definitions"
			}
			id = ComponentID{Section: section, Name: value.Value}
		}
		return g.addEdge(owner, id, value.Value, valuePath)
	}
	if mapping := mapValue(node, "mapping"); mapping != nil && mapping.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(mapping.Content); i += 2 {
			if err := add(mapping.Content[i+1], appendPath(appendPath(path, "mapping"), mapping.Content[i].Value)); err != nil {
				return err
			}
		}
	}
	if value := mapValue(node, "defaultMapping"); value != nil {
		return add(value, appendPath(path, "defaultMapping"))
	}
	return nil
}

func (g *ComponentGraph) addSecurityEdges(node *yaml.Node, owner *ComponentID, swagger bool, path []string) error {
	security := mapValue(node, "security")
	if security == nil || security.Kind != yaml.SequenceNode {
		return nil
	}
	section := "securitySchemes"
	if swagger {
		section = "securityDefinitions"
	}
	for i, requirement := range security.Content {
		if requirement.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(requirement.Content); j += 2 {
			name := requirement.Content[j].Value
			target := ComponentID{Section: section, Name: name}
			if err := g.addEdge(owner, target, name, appendPath(appendPath(path, "security"), fmt.Sprintf("%d.%s", i, name))); err != nil {
				return err
			}
		}
	}
	return nil
}

func openAPIComponentSectionSet(version string) map[string]bool {
	sections := make(map[string]bool, len(baseOpenAPIComponentSections)+2)
	for _, section := range baseOpenAPIComponentSections {
		sections[section] = true
	}
	if strings.HasPrefix(version, "3.1") || strings.HasPrefix(version, "3.2") {
		sections["pathItems"] = true
	}
	if strings.HasPrefix(version, "3.2") {
		sections["mediaTypes"] = true
	}
	return sections
}

func isSwaggerComponentSection(value string) bool {
	switch value {
	case "definitions", "parameters", "responses", "securityDefinitions":
		return true
	default:
		return false
	}
}

func isSecurityRequirementOwner(path []string, owner *ComponentID) bool {
	if len(path) == 0 {
		return false
	}
	if owner != nil && owner.Section != "pathItems" && owner.Section != "callbacks" {
		return false
	}
	if owner == nil && path[0] != "paths" && path[0] != "webhooks" {
		return false
	}
	for _, part := range path {
		if strings.HasPrefix(part, "x-") || part == "examples" {
			return false
		}
	}
	last := path[len(path)-1]
	if standardOperationKeys[last] || last == "query" {
		return true
	}
	return len(path) > 1 && path[len(path)-2] == "additionalOperations"
}

func isLinkObjectPath(path []string) bool {
	if len(path) == 3 && path[0] == "components" && path[1] == "links" {
		return true
	}
	if len(path) < 4 || path[len(path)-2] != "links" {
		return false
	}
	responsePath := path[:len(path)-2]
	if len(responsePath) == 3 && responsePath[0] == "components" && responsePath[1] == "responses" {
		return true
	}
	if len(responsePath) < 3 || responsePath[len(responsePath)-2] != "responses" {
		return false
	}
	operationPath := responsePath[:len(responsePath)-2]
	last := operationPath[len(operationPath)-1]
	if standardOperationKeys[last] || last == "query" {
		return true
	}
	return len(operationPath) > 1 && operationPath[len(operationPath)-2] == "additionalOperations"
}

func walkYAML(node *yaml.Node, fn func(*yaml.Node)) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		fn(node)
	}
	for _, child := range node.Content {
		walkYAML(child, fn)
	}
}

func walkYAMLNodes(node *yaml.Node, fn func(*yaml.Node)) {
	fn(node)
	for _, child := range node.Content {
		walkYAMLNodes(child, fn)
	}
}
