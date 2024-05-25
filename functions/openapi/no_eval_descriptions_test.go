package openapi

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"regexp"
	"testing"
)

func TestNoEvalInDescriptions_GetSchema(t *testing.T) {
	def := NoEvalInDescriptions{}
	assert.Equal(t, "noEvalDescription", def.GetSchema().Name)
}

func TestNoEvalInDescriptions_RunRule(t *testing.T) {
	def := NoEvalInDescriptions{}
	res := def.RunRule(nil, model.RuleFunctionContext{})
	assert.Len(t, res, 0)
}

func TestNoEvalInDescriptions_RunRule_SuccessCheck(t *testing.T) {

	yml := `paths:
  /pizza/:
    description: do you do crisps?"
  /cake/:
    description: nah mate, only onions.
components:
  schemas:
    CrispsOnion:
      description: a lovely bunch of coconuts`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_eval_description", "", nil)
	fo := make(map[string]string)
	fo["pattern"] = "eval\\("
	comp, _ := regexp.Compile(fo["pattern"])
	rule.PrecompiledPattern = comp
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), fo)
	ctx.Rule = &rule
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)

	def := NoEvalInDescriptions{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 0)
}

func TestNoEvalInDescriptions_RunRule_EvalFail(t *testing.T) {

	yml := `paths:
  /pizza/:
    description: eval("alert('hax0r')")"
  /cake/:
    description: nah mate, only onions.
components:
  schemas:
    CrispsOnion:
      description: eval("/*scripkiddy.js*/")`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_eval_description", "", nil)
	fo := make(map[string]string)
	fo["pattern"] = "eval\\("
	comp, _ := regexp.Compile(fo["pattern"])
	rule.PrecompiledPattern = comp

	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), fo)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.Rule = &rule

	def := NoEvalInDescriptions{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 2)
}

func TestNoScriptInDescriptions_RunRule_EvalFail(t *testing.T) {

	yml := `paths:
  /pizza/:
    description: <script>console.log('hax0r')</script>"
  /cake/:
    description: nah mate, only onions.
components:
  schemas:
    CrispsOnion:
      description: no hack`

	path := "$"

	var rootNode yaml.Node
	mErr := yaml.Unmarshal([]byte(yml), &rootNode)
	assert.NoError(t, mErr)

	nodes, _ := utils.FindNodes([]byte(yml), path)

	rule := buildOpenApiTestRuleAction(path, "no_script_description", "", nil)

	fo := make(map[string]string)
	fo["pattern"] = "<script"
	comp, _ := regexp.Compile(fo["pattern"])
	rule.PrecompiledPattern = comp
	ctx := buildOpenApiTestContext(model.CastToRuleAction(rule.Then), fo)
	config := index.CreateOpenAPIIndexConfig()
	ctx.Index = index.NewSpecIndexWithConfig(&rootNode, config)
	ctx.Rule = &rule

	def := NoEvalInDescriptions{}
	res := def.RunRule(nodes, ctx)

	assert.Len(t, res, 1)
}
