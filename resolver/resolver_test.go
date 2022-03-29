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

func Benchmark_ResolveDocumentStripe(b *testing.B) {
	stripe, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	for n := 0; n < b.N; n++ {
		var rootNode yaml.Node
		yaml.Unmarshal(stripe, &rootNode)
		index := model.NewSpecIndex(&rootNode)
		resolver := NewResolver(index)
		resolver.Resolve()
	}
}

func TestResolver_ResolveComponents_CircularSpec(t *testing.T) {

	circular, _ := ioutil.ReadFile("../model/test_files/circular-tests.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(circular, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 3)

	_, err := yaml.Marshal(resolver.resolvedRoot)
	assert.NoError(t, err)
}

func TestResolver_ResolveComponents_Stripe(t *testing.T) {

	stripe, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(stripe, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)

}

func TestResolver_ResolveComponents_MixedRef(t *testing.T) {

	mixedref, _ := ioutil.ReadFile("../model/test_files/mixedref-burgershop.openapi.yaml")
	var rootNode yaml.Node
	yaml.Unmarshal(mixedref, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)

}

func TestResolver_ResolveComponents_k8s(t *testing.T) {

	k8s, _ := ioutil.ReadFile("../model/test_files/k8s.json")
	var rootNode yaml.Node
	yaml.Unmarshal(k8s, &rootNode)

	index := model.NewSpecIndex(&rootNode)

	resolver := NewResolver(index)
	assert.NotNil(t, resolver)

	circ := resolver.Resolve()
	assert.Len(t, circ, 0)
}
