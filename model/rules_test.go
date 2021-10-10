package model

import (
	"github.com/daveshanley/vaccum/parser"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestRuleSchema(t *testing.T) {

	//schema, err := ioutil.ReadFile("schemas/ruleset.schema.json")
	//assert.NoError(t, err)
	//
	//goodRules, err := ioutil.ReadFile("../test_files/rules.json")
	//assert.NoError(t, err)

	schemaFile = "file://"

	r, err := parser.ValidateJSONAgainstSchema(schema, goodRules)
	assert.NoError(t, err)

	assert.True(t, r.Valid())

}
