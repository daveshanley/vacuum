package cmd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
	"io"
	"os"
	"testing"
)

func TestGenerateIgnoreFileCommand(t *testing.T) {
	outputFile := "/tmp/ignorefile.yaml"

	cmd := GetGenerateIgnoreFileCommand()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetArgs([]string{
		"./test_data/vacuum-report.json",
		outputFile,
	})
	cmdErr := cmd.Execute()
	outBytes, err := io.ReadAll(b)

	assert.NoError(t, cmdErr)
	assert.NoError(t, err)
	assert.NotNil(t, outBytes)

	defer os.Remove(outputFile)

	ignoreFileBytes, err := os.ReadFile(outputFile)
	assert.NoError(t, err)

	ignorefileValues := map[string][]string{}
	err = yaml.Unmarshal(ignoreFileBytes, &ignorefileValues)
	assert.NoError(t, err)

	expectedIgnorefileValues := map[string][]string{
		"oas3-missing-example": {
			"$.components.schemas['Error']",
			"$.components.schemas['Burger']",
			"$.components.schemas['Fries']",
			"$.components.schemas['Fries'].properties['seasoning']",
			"$.components.schemas['Dressing']",
			"$.components.schemas['Drink']",
		},
		"oas3-unused-component": {
			"$.components.schemas['Error']",
			"$.components.schemas['Burger']",
			"$.components.schemas['Fries']",
			"$.components.schemas['Dressing']",
			"$.components.schemas['Drink']",
		},
	}
	assert.Equal(t, expectedIgnorefileValues, ignorefileValues)

}
