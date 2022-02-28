package motor

import (
	"fmt"
	"github.com/daveshanley/vacuum/functions"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/utils"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
	"sync"
)

// ApplyRules will apply a loaded model.RuleSet against an OpenAPI specification.
func ApplyRules(ruleSet *model.RuleSet, spec []byte) ([]model.RuleFunctionResult, error) {

	builtinFunctions := functions.MapBuiltinFunctions()
	var ruleResults []model.RuleFunctionResult

	var ruleWaitGroup sync.WaitGroup
	ruleWaitGroup.Add(len(ruleSet.Rules))

	var specResolved yaml.Node
	var specUnresolved yaml.Node
	err := yaml.Unmarshal(spec, &specResolved)
	if err != nil {
		return nil, err
	}
	yaml.Unmarshal(spec, &specUnresolved)

	resolved, errs := model.ResolveOpenAPIDocument(&specResolved)

	for _, er := range errs {
		fmt.Printf("Resolving Issue: %v (%d: %d)\n", er.Error, er.Node.Line, er.Node.Column)
	}

	var errors []error
	for _, rule := range ruleSet.Rules {
		ruleSpec := resolved
		if !rule.Resolved {
			ruleSpec = &specUnresolved
		}
		go runRule(rule, ruleSpec, builtinFunctions, &ruleResults, &ruleWaitGroup, &errors)
	}

	ruleWaitGroup.Wait()
	// did something go wrong?

	return ruleResults, nil
}

func runRule(rule *model.Rule, specNode *yaml.Node, builtinFunctions functions.Functions,
	ruleResults *[]model.RuleFunctionResult, wg *sync.WaitGroup, errors *[]error) {

	defer wg.Done()
	var givenPaths []string
	if x, ok := rule.Given.(string); ok {
		givenPaths = append(givenPaths, x)
	}

	if x, ok := rule.Given.([]interface{}); ok {
		for _, gpI := range x {
			if gp, ok := gpI.(string); ok {
				givenPaths = append(givenPaths, gp)
			}
			if gp, ok := gpI.(int); ok {
				givenPaths = append(givenPaths, fmt.Sprintf("%v", gp))
			}
		}

	}

	for _, givenPath := range givenPaths {

		nodes, err := utils.FindNodesWithoutDeserializing(specNode, givenPath)
		if err != nil {
			*errors = append(*errors, err)
			return
		}
		if len(nodes) <= 0 {
			continue
		}

		var ruleAction model.RuleAction
		err = mapstructure.Decode(rule.Then, &ruleAction)

		if err == nil {

			ruleResults = buildResults(rule, builtinFunctions, ruleAction, ruleResults, nodes)

		} else {
			var ruleActions []model.RuleAction
			err = mapstructure.Decode(rule.Then, &ruleActions)

			if err == nil {
				for _, rAction := range ruleActions {
					ruleResults = buildResults(rule, builtinFunctions, rAction, ruleResults, nodes)
				}
			}
		}
	}
}

var lock sync.Mutex

func buildResults(rule *model.Rule, builtinFunctions functions.Functions, ruleAction model.RuleAction,
	ruleResults *[]model.RuleFunctionResult, nodes []*yaml.Node) *[]model.RuleFunctionResult {

	ruleFunction := builtinFunctions.FindFunction(ruleAction.Function)

	if ruleFunction != nil {

		rfc := model.RuleFunctionContext{
			Options:    ruleAction.FunctionOptions,
			RuleAction: &ruleAction,
			Rule:       rule,
			Given:      rule.Given,
		}

		// validate the rule is configured correctly before running it.
		res, errs := model.ValidateRuleFunctionContextAgainstSchema(ruleFunction, rfc)
		if !res {
			for _, e := range errs {
				lock.Lock()
				*ruleResults = append(*ruleResults, model.RuleFunctionResult{Message: e})
				lock.Unlock()
			}
		} else {

			// iterate through nodes and supply them one at a time so we don't pollute each run
			// TODO: change this signature to be singular and not an array so this is handled permanently.

			for _, node := range nodes {
				runRuleResults := ruleFunction.RunRule([]*yaml.Node{node}, rfc)

				// because this function is running in multiple threads, we need to sync access to the final result
				// list, otherwise things can get a bit random.
				lock.Lock()
				*ruleResults = append(*ruleResults, runRuleResults...)
				lock.Unlock()
			}

		}
	} else {

		// TODO: Fix this error handling.
		//fmt.Printf("oooo nice, an error here.")
	}
	return ruleResults
}
