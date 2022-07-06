package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"testing"
)

func TestGetLintCommand(t *testing.T) {
	cmd := GetLintCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"../model/test_files/burgershop.openapi.yaml"})
	cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	// assert.Error(t, cmdErr) // need return code to be 1 first, disabling for now.
}

func TestGetLintCommand_Ruleset(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/custom-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	_ = cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	// assert.NoError(t, cmdErr) // need return code to be 1 first, disabling for now.
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_RulesetMissing(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-r",
		"../rulesets/examples/nope.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)
	assert.Error(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_NoRules(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_NoSpec(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.Error(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_BadSpec(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/not-here-not-there.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.Error(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_BadRuleset(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-r",
		"../model/test_files/burgershop.openapi.yaml", // not a ruleset.
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.Error(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_InvalidRuleset(t *testing.T) {

	json := `{
  "documentationUrl": "quobix.com",
  "rules": {
    "length-test-description": {
      "description": "this is an invalid rule def, because the JSONPath is borked",
      "recommended": true,
      "type": "style",
      "given": "I AM NOT A PATH <-- ",
      "severity": "error",
      "then": {
        "function": "length",
		"field": "required",
		"functionOptions" : { 
			"max" : "2"
		}
      }
    }
  }
}`

	tmp, _ := ioutil.TempFile("", "")
	defer tmp.Close()
	_, _ = io.WriteString(tmp, json)

	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-r",
		tmp.Name(),
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.Error(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_SpecificRules(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Examples(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"examples",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Descriptions(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"descriptions",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Info(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"information",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Schemas(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"schemas",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Security(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"security",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Validation(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"validation",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Operations(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"operations",
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Tags(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"tags",
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Category_Default(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-c",
		"nope",
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Details_NoCat(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Details_NoCat_NotSilent(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-d",
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Details_NoCat_Snippets(t *testing.T) {
	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-s",
		"-r",
		"../rulesets/examples/specific-ruleset.yaml",
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Details_ErrorOverride(t *testing.T) {

	yaml := `extends: [[spectral:oas, recommended]]
rules:
  oas3-valid-schema-example: error`

	tmp, _ := ioutil.TempFile("", "")
	defer tmp.Close()
	_, _ = io.WriteString(tmp, yaml)

	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-x",
		"-d",
		"-r",
		tmp.Name(),
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetLintCommand_Details_Snippets(t *testing.T) {

	yaml := `extends: [[spectral:oas, off]]
rules:
  oas3-valid-schema-example: true`

	tmp, _ := ioutil.TempFile("", "")
	defer tmp.Close()
	_, _ = io.WriteString(tmp, yaml)

	cmd := GetLintCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-s",
		"-d",
		"-r",
		tmp.Name(),
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}
