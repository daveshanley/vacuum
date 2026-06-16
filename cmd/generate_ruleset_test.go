package cmd

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestGenerateRulesetCommand(t *testing.T) {
	cmd := GetGenerateRulesetCommand()
	b := bytes.NewBufferString("")
	outputPrefix := filepath.Join(t.TempDir(), "test-output")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"all",
		outputPrefix,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	requireSingleGeneratedFile(t, outputPrefix+"-all.yaml")
}

func TestGenerateRulesetCommand_Recommended(t *testing.T) {
	cmd := GetGenerateRulesetCommand()
	b := bytes.NewBufferString("")
	outputPrefix := filepath.Join(t.TempDir(), "test-output")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"recommended",
		outputPrefix,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	requireSingleGeneratedFile(t, outputPrefix+"-recommended.yaml")
}

func TestGenerateRulesetCommand_InvalidType(t *testing.T) {
	cmd := GetGenerateRulesetCommand()
	b := bytes.NewBufferString("")
	outputPrefix := filepath.Join(t.TempDir(), "test-output")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"fish-cakes",
		outputPrefix,
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGenerateRulesetCommand_NoArgs(t *testing.T) {
	cmd := GetGenerateRulesetCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}

func TestGenerateRulesetCommand_BadWrite(t *testing.T) {
	cmd := GetGenerateRulesetCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"recommended",
		"/no/no/no-stop-/",
	})
	cmdErr := cmd.Execute()
	assert.Error(t, cmdErr)
}
