package resolver

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"testing"
)

func TestNewResolver(t *testing.T) {
	assert.Nil(t, NewResolver(nil))
}

func TestResolver_ResolveComponents_CircularSpec(t *testing.T) {

	asana, _ := ioutil.ReadFile("../model/test_files/circular-tests.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.ResolveComponents()
	assert.Len(t, circ, 2)

}

func TestResolver_ResolveComponents_Stripe(t *testing.T) {

	asana, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.ResolveComponents()
	assert.Len(t, circ, 0)

}

func TestResolver_ResolveComponents_k8s(t *testing.T) {

	asana, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(asana, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.ResolveComponents()
	assert.Len(t, circ, 0)

}
