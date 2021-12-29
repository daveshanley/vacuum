package openapi_functions

import (
	"github.com/daveshanley/vacuum/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestGetTagsFromRoot(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	assert.Len(t, GetTagsFromRoot(nodes), 2)
}

func TestGetTagsFromRoot_Fail(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$.does-not-exist")
	assert.Len(t, GetTagsFromRoot(nodes), 0)
}

func TestGetOperationsFromRoot(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$")
	assert.Len(t, GetOperationsFromRoot(nodes), 10) // this is 5 paths and sequential map nodes.
}

func TestGetOperationsFromRoot_Fail(t *testing.T) {
	sampleYaml, _ := ioutil.ReadFile("../../model/test_files/burgershop.openapi.yaml")
	nodes, _ := utils.FindNodes(sampleYaml, "$.made-up-nothing")
	assert.Len(t, GetOperationsFromRoot(nodes), 0)
}

func TestGetAllOperationsJSONPath(t *testing.T) {
	assert.NotNil(t, GetAllOperationsJSONPath())
}
