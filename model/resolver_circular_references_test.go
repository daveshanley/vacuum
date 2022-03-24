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

func TestMagicJourney_ExtractRefs(t *testing.T) {

	//stripe, _ := ioutil.ReadFile("test_files/asana.yaml")
	stripe, _ := ioutil.ReadFile("test_files/stripe.yaml")
	//stripe, _ := ioutil.ReadFile("../petstore-fixed.json")
	//stripe, _ := ioutil.ReadFile("../petstore.json")
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
	assert.Equal(t, 76, index.componentsInlineDuplicateCount)
	assert.Equal(t, 67, index.componentsInlineUniqueCount)

}
