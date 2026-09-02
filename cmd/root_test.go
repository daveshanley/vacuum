package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type failingSliceValue struct {
	replace bool
}

func (f *failingSliceValue) Set(string) error    { return nil }
func (f *failingSliceValue) String() string      { return "" }
func (f *failingSliceValue) Type() string        { return "stringArray" }
func (f *failingSliceValue) Append(string) error { return nil }
func (f *failingSliceValue) GetSlice() []string  { return nil }
func (f *failingSliceValue) Replace([]string) error {
	if f.replace {
		return errors.New("replace failed")
	}
	return nil
}

type nonSliceCollectionValue struct{}

func (*nonSliceCollectionValue) Set(string) error { return nil }
func (*nonSliceCollectionValue) String() string   { return "" }
func (*nonSliceCollectionValue) Type() string     { return "stringArray" }

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

	flag := &pflag.Flag{Name: "fake", Value: &nonSliceCollectionValue{}}
	assert.ErrorContains(t, setBoundFlag(pflag.NewFlagSet("test", pflag.ContinueOnError), flag, []string{"x"}), "without slice support")
	flag.Value = &failingSliceValue{replace: true}
	assert.ErrorContains(t, setBoundFlag(pflag.NewFlagSet("test", pflag.ContinueOnError), flag, []string{"x"}), "replace failed")
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
