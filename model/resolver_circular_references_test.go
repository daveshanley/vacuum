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
