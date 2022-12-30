package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
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
	outBytes, err := io.ReadAll(b)

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
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	defer os.Remove("blue-shoes.json")
}

func TestGetSpectralReportCommand_StdInOut(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"-i", "-o"})
	cmd.SetIn(strings.NewReader("openapi: 3.1.0"))
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}

func TestGetSpectralReportCommand_StdInOutNoPretty(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{"-i", "-o", "-n"})
	cmd.SetIn(strings.NewReader("openapi: 3.1.0"))
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
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
	outBytes, err := io.ReadAll(b)

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

func TestGetSpectralReportCommand_BadWrite(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"/cant-write-here/ok/no.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetSpectralReportCommand_WrongFile(t *testing.T) {
	cmd := GetSpectralReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../rulesets/examples/all-ruleset.yaml",
	})
	cmdErr := cmd.Execute()
	assert.NoError(t, cmdErr)
	defer os.Remove("vacuum-spectral-report.json")
}

func TestGetSpectralReportCommand_BadRuleset_WrongFile(t *testing.T) {
	cmd := GetSpectralReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../model/test_files/petstorev3.json",
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
