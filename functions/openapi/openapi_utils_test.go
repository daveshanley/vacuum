package openapi_functions

import (
	"github.com/daveshanley/vaccum/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestGetTagsFromRoot(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	assert.Len(t, GetTagsFromRoot(nodes), 2)
}
func TestGetOperationsFromRoot(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	assert.Len(t, GetOperationsFromRoot(nodes), 10) // this is 5 paths and sequential map nodes.
}
