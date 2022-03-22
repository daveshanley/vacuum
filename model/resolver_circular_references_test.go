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

	mj := new(MagicJourney)

	stripe, _ := ioutil.ReadFile("test_files/asana.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	mj.allRefs = make(map[string]*Reference)
	mj.allMappedRefs = make(map[string]*Reference)
	mj.pathRefs = make(map[string]map[string]*Reference)
	mj.paramOpRefs = make(map[string]map[string]*Reference)
	mj.paramCompRefs = make(map[string]*Reference)

	mj.root = &rootNode

	results := mj.ExtractRefs(mj.root)
	assert.Len(t, results, 171)

	extracted := mj.ExtractComponentsFromRefs(results)
	assert.Len(t, extracted, 171)

	assert.Equal(t, 118, mj.GetPathCount())
	assert.Equal(t, 152, mj.GetOperationCount())

	pcount, err := mj.GetParameterCount()
	// TODO: continue the magic journey.

	assert.Equal(t, 0, pcount)
	assert.NoError(t, err)

}
