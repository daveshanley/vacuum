package model

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

// working code so far.
func TestCheckForSchemaCircularReferences(t *testing.T) {

	//stripe, _ := ioutil.ReadFile("test_files/stripe.yaml")
	stripe, _ := ioutil.ReadFile("test_files/circular-tests.yaml")

	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	results := CheckForSchemaCircularReferences(&rootNode)

	assert.NotNil(t, results)
	assert.Len(t, results, 2)

	for _, result := range results {
		assert.Equal(t, result.Journey[len(result.Journey)-1].Definition, result.LoopPoint.Definition)
	}
}
