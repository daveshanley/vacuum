package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

func TestBindFlagsStringCollections(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  []string
	}{
		{"string slice", []string{"public", "partner"}, []string{"public", "partner"}},
		{"interface slice", []any{"public", "partner"}, []string{"public", "partner"}},
		{"CSV environment shape", `public,"partner,shared"`, []string{"public", "partner,shared"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.StringArray("include-tag", nil, "")
			tree := viper.New()
			tree.Set("include-tag", tc.value)
			require.NoError(t, bindFlags(flags, tree))
			actual, err := flags.GetStringArray("include-tag")
			require.NoError(t, err)
			assert.Equal(t, tc.want, actual)
			assert.True(t, flags.Changed("include-tag"))
		})
	}
}

func TestBindFlagsCollectionErrorsAndScalar(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringArray("include-tag", nil, "")
	tree := viper.New()
	tree.Set("include-tag", 42)
	assert.ErrorContains(t, bindFlags(flags, tree), "expected a string collection")
	_, err := stringCollectionValues([]any{"public", 42})
	assert.ErrorContains(t, err, "item at index 1")

	flags = pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("mode", "", "")
	tree = viper.New()
	tree.Set("mode", "all")
	require.NoError(t, bindFlags(flags, tree))
	mode, err := flags.GetString("mode")
	require.NoError(t, err)
	assert.Equal(t, "all", mode)

	_, err = stringCollectionValues(`"unterminated`)
	assert.ErrorContains(t, err, "invalid string collection")

}

func TestBindEnvironmentFlags(t *testing.T) {
	t.Setenv("VACUUM_INCLUDE_TAG", "public,partner")
	t.Setenv("VACUUM_TAG_MATCH", "all")
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringArray("include-tag", nil, "")
	flags.String("tag-match", "any", "")
	require.NoError(t, bindEnvironmentFlags(flags))
	tags, err := flags.GetStringArray("include-tag")
	require.NoError(t, err)
	assert.Equal(t, []string{"public", "partner"}, tags)
	mode, err := flags.GetString("tag-match")
	require.NoError(t, err)
	assert.Equal(t, "all", mode)

	require.NoError(t, flags.Set("tag-match", "any"))
	require.NoError(t, os.Setenv("VACUUM_TAG_MATCH", "all"))
	require.NoError(t, bindEnvironmentFlags(flags))
	mode, err = flags.GetString("tag-match")
	require.NoError(t, err)
	assert.Equal(t, "any", mode)
}
