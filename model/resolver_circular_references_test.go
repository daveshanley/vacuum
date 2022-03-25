package model

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

// working code so far.
func TestCheckForSchemaCircularReferences(t *testing.T) {

	circularTest, _ := ioutil.ReadFile("test_files/circular-tests.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(circularTest, &rootNode)

	results, ko, seq := CheckForSchemaCircularReferences("$.components.schemas", &rootNode)

	assert.NotNil(t, results)
	assert.Len(t, results, 3)
	assert.Len(t, ko, 9)
	assert.Len(t, seq, 9)

	for _, result := range results {
		assert.Equal(t, result.Journey[len(result.Journey)-1].Definition, result.LoopPoint.Definition)
	}
}

func TestCheckForSchemaCircularReferences_Stripe(t *testing.T) {

	stripe, _ := ioutil.ReadFile("test_files/stripe.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	results, _, _ := CheckForSchemaCircularReferences("$.components.schemas",
		&rootNode)
	assert.Nil(t, results)

}

func TestSpecIndex_ExtractRefsStripe(t *testing.T) {

	stripe, _ := ioutil.ReadFile("test_files/stripe.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 537)
	assert.Len(t, index.allMappedRefs, 537)
	assert.Equal(t, 246, index.pathCount)
	assert.Equal(t, 402, index.operationCount)
	assert.Equal(t, 537, index.schemaCount)
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 143, index.operationParamCount)
	assert.Equal(t, 76, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 67, index.componentsInlineParamUniqueCount)

}

func TestSpecIndex_Asana(t *testing.T) {

	asana, _ := ioutil.ReadFile("test_files/asana.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 171)
	assert.Len(t, index.allMappedRefs, 171)
	assert.Equal(t, 118, index.pathCount)
	assert.Equal(t, 152, index.operationCount)
	assert.Equal(t, 135, index.schemaCount)
	assert.Equal(t, 26, index.globalTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 30, index.componentParamCount)
	assert.Equal(t, 107, index.operationParamCount)
	assert.Equal(t, 8, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 69, index.componentsInlineParamUniqueCount)
}

func TestSpecIndex_k8s(t *testing.T) {

	asana, _ := ioutil.ReadFile("test_files/k8s.json")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 563)
	assert.Len(t, index.allMappedRefs, 563)
	assert.Equal(t, 436, index.pathCount)
	assert.Equal(t, 853, index.operationCount)
	assert.Equal(t, 563, index.schemaCount)
	assert.Equal(t, 0, index.globalTagsCount)
	assert.Equal(t, 58, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 36, index.operationParamCount)
	assert.Equal(t, 26, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 10, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 58, index.GetTotalTagsCount())

}

func TestSpecIndex_PetstoreV2(t *testing.T) {

	asana, _ := ioutil.ReadFile("test_files/petstorev2.json")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 6)
	assert.Len(t, index.allMappedRefs, 6)
	assert.Equal(t, 14, index.pathCount)
	assert.Equal(t, 20, index.operationCount)
	assert.Equal(t, 6, index.schemaCount)
	assert.Equal(t, 3, index.globalTagsCount)
	assert.Equal(t, 3, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 1, index.componentParamCount)
	assert.Equal(t, 11, index.operationParamCount)
	assert.Equal(t, 4, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 7, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 3, index.GetTotalTagsCount())
}

func TestSpecIndex_PetstoreV3(t *testing.T) {

	asana, _ := ioutil.ReadFile("test_files/petstorev3.json")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 7)
	assert.Len(t, index.allMappedRefs, 7)
	assert.Equal(t, 13, index.pathCount)
	assert.Equal(t, 19, index.operationCount)
	assert.Equal(t, 8, index.schemaCount)
	assert.Equal(t, 3, index.globalTagsCount)
	assert.Equal(t, 3, index.operationTagsCount)
	assert.Equal(t, 0, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 9, index.operationParamCount)
	assert.Equal(t, 3, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 6, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 3, index.GetTotalTagsCount())

}

func TestSpecIndex_BurgerShop(t *testing.T) {

	asana, _ := ioutil.ReadFile("test_files/burgershop.openapi.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := NewSpecIndex(&rootNode)

	assert.Len(t, index.allRefs, 4)
	assert.Len(t, index.allMappedRefs, 4)
	assert.Equal(t, 5, index.pathCount)
	assert.Equal(t, 5, index.operationCount)
	assert.Equal(t, 5, index.schemaCount)
	assert.Equal(t, 2, index.globalTagsCount)
	assert.Equal(t, 2, index.operationTagsCount)
	assert.Equal(t, 2, index.globalLinksCount)
	assert.Equal(t, 0, index.componentParamCount)
	assert.Equal(t, 2, index.operationParamCount)
	assert.Equal(t, 1, index.componentsInlineParamDuplicateCount)
	assert.Equal(t, 1, index.componentsInlineParamUniqueCount)
	assert.Equal(t, 2, index.GetTotalTagsCount())

}
