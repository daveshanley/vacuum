package cmd

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
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
	outBytes, err := ioutil.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	time := time.Now()
	file := fmt.Sprintf("vacuum-report-%s.json", time.Format("01-02-06-15_04_05"))
	defer os.Remove(file)
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
	outBytes, err := ioutil.ReadAll(b)

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
	outBytes, err := ioutil.ReadAll(b)

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
	outBytes, err := ioutil.ReadAll(b)

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
