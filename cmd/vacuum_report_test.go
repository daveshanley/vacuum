package cmd

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestGetVacuumReportCommand(t *testing.T) {
	cmd := GetVacuumReportCommand()
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

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_StdInOut(t *testing.T) {
	cmd := GetVacuumReportCommand()
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

func TestGetVacuumReportCommand_NoPretty(t *testing.T) {
	cmd := GetVacuumReportCommand()
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

func TestGetVacuumReportCommand_Compress(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-c",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json.gz", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_CustomPrefix(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"cheesy-shoes",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("cheesy-shoes-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_WithRuleSet(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../rulesets/examples/norules-ruleset.yaml",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
}

func TestGetVacuumReportCommand_WithBadRuleset(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"I do not exist",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_WithBadRuleset_WrongFile(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"-r",
		"../model/test_files/petstorev3.json",
		"../model/test_files/petstorev3.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_BadWrite(t *testing.T) {
	cmd := GetVacuumReportCommand()
	// global flag exists on root only.
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")

	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"../model/test_files/petstorev3.json",
		"/cant-write-here/oh-noes.json",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGetVacuumReportCommand_NoArgs(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)

}

func TestGetVacuumReportCommand_BadFile(t *testing.T) {
	cmd := GetVacuumReportCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"I do not exist",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)

}

func TestGetVacuumReport_WithIgnoreFile(t *testing.T) {

	yaml := `
extends: [[vacuum:oas, recommended]]
rules:
    url-starts-with-major-version:
        description: Major version must be the first URL component
        message: All paths must start with a version number, eg /v1, /v2
        given: $.paths
        severity: error
        then:
            function: pattern
            functionOptions:
                match: "/v[0-9]+/"
`

	tmp, _ := os.CreateTemp("", "")
	_, _ = io.WriteString(tmp, yaml)

	defer os.Remove(tmp.Name())

	// capture output for testing - no longer needed since we don't use pterm
	b := bytes.NewBufferString("")

	cmd := GetVacuumReportCommand()
	cmd.PersistentFlags().StringP("ruleset", "r", "", "")
	cmd.SetArgs([]string{
		"--ignore-file",
		"../model/test_files/burgershop.ignorefile.yaml",
		"-r",
		tmp.Name(),
		"../model/test_files/burgershop.openapi.yaml",
	})
	cmdErr := cmd.Execute()
	assert.NoError(t, cmdErr)
}
