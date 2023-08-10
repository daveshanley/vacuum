package cmd

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNonExistingConfigFile(t *testing.T) {
	b := bytes.NewBufferString("")
	rootCmd := GetRootCommand()
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"lint", "../model/test_files/burgershop.openapi.yaml", "--config=/a/non/existing/config/file/path"})
	exErr := rootCmd.Execute()
	assert.Error(t, exErr)
}
func TestValidConfigFile(t *testing.T) {
	b := bytes.NewBufferString("")
	rootCmd := GetRootCommand()
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"lint", "../model/test_files/burgershop.openapi.yaml", "--config=../model/test_files/vacuum-global.conf.yaml"})
	exErr := rootCmd.Execute()
	assert.NoError(t, exErr)
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
}
func TestGlobalFlagConfigFile(t *testing.T) {
	b := bytes.NewBufferString("")
	rootCmd := GetRootCommand()
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"lint", "../model/test_files/burgershop.openapi.yaml", "--config=../model/test_files/vacuum-global.conf.yaml"})
	exErr := rootCmd.Execute()
	assert.NoError(t, exErr)
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	//TODO test global flag override
}
func TestLocalFlagConfigFile(t *testing.T) {
	b := bytes.NewBufferString("")
	rootCmd := GetRootCommand()
	rootCmd.SetOut(b)
	rootCmd.SetArgs([]string{"lint", "../model/test_files/burgershop.openapi.yaml", "--config=../model/test_files/vacuum-local.conf.yaml"})
	exErr := rootCmd.Execute()
	assert.NoError(t, exErr)
	outBytes, err := io.ReadAll(b)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)
	//TODO test local flag override
}
