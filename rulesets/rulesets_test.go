package rulesets

import (
	"github.com/daveshanley/vacuum/motor"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestBuildDefaultRuleSets(t *testing.T) {

	rs := BuildDefaultRuleSets()
	assert.NotNil(t, rs.GenerateOpenAPIDefaultRuleSet())
	assert.Len(t, rs.GenerateOpenAPIDefaultRuleSet().Rules, 30)

}

func TestStripeSpecAgainstDefaultRuleSet(t *testing.T) {

	b, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	rs := BuildDefaultRuleSets()
	results, err := motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), b)

	assert.NoError(t, err)
	assert.NotNil(t, results)

}

func Benchmark_StripeSpecAgainstDefaultRuleSet(b *testing.B) {
	m, _ := ioutil.ReadFile("../model/test_files/stripe.yaml")
	rs := BuildDefaultRuleSets()
	for n := 0; n < b.N; n++ {
		motor.ApplyRules(rs.GenerateOpenAPIDefaultRuleSet(), m)
	}
}
