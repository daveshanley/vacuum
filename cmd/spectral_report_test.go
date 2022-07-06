package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestGetSpectralReportCommand(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("vacuum-spectral-report.json")
}

func TestGetSpectralReportCommand_CustomName(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"blue-shoes.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("blue-shoes.json")
}

func TestGetSpectralReportCommand_CustomRuleset(t *testing.T) {
	cmd := GetSpectralReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/petstorev3.json",
		"blue-shoes.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("blue-shoes.json")
}

func TestGetSpectralReportCommand_BadRuleset(t *testing.T) {
	cmd := GetSpectralReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"I do not exist",
		"../model/test_files/petstorev3.json",
		"blue-shoes.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetSpectralReportCommand_BadInput(t *testing.T) {
	cmd := GetSpectralReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"I do not exist",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetSpectralReportCommand_NoArgs(t *testing.T) {
	cmd := GetSpectralReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}
