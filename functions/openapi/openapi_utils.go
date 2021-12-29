package openapi_functions

import (
	"github.com/daveshanley/vaccum/utils"
	"gopkg.in/yaml.v3"
)

func GetAllOperationsJSONPath() string {
	return "$.paths[*]['get','put','post','delete','options','head','patch','trace']"
}

func GetTagsFromRoot(nodes []*yaml.Node) []*yaml.Node {
	for _, node := range nodes {
		_, tags := utils.FindFirstKeyNode("tags", node.Content)
		if len(tags.Content) > 0 {
			return tags.Content
		}
	}
	return nil
}

func GetOperationsFromRoot(nodes []*yaml.Node) []*yaml.Node {
	for _, node := range nodes {
		_, tags := utils.FindFirstKeyNode("paths", node.Content)
		if len(tags.Content) > 0 {
			return tags.Content
		}
	}
	return nil
}
