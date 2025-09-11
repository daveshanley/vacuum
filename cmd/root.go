// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFile string
	Version    string
	Commit     string
	Date       string
)

func Execute(version, commit, date string) {
	Version = version
	Commit = commit
	Date = date
	if err := GetRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func GetRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "vacuum lint <your-openapi-file.yaml>",
		Short: "vacuum is a very fast OpenAPI linter",
		Long:  `vacuum is a very fast OpenAPI linter. It will suck all the lint off your spec in milliseconds`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := useConfigFile(cmd)
			if err != nil {
				pterm.Error.Printf("%s", err)
			}
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			PrintBanner()
			pterm.Println(">> Welcome! To lint something, try 'vacuum lint <my-openapi-spec.yaml>'")
			pterm.Println()
			pterm.Println("To see all the options, try 'vacuum --help'")
			pterm.Println()
			return nil
		},
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (defaults to ./vacuum.conf.yaml) ")
	rootCmd.PersistentFlags().BoolP("time", "t", false, "Show how long vacuum took to run")
	rootCmd.PersistentFlags().StringP("ruleset", "r", "", "Location of a vacuum (or Spectral) ruleset")
	rootCmd.PersistentFlags().StringP("functions", "f", "", "Path to custom functions")
	rootCmd.PersistentFlags().StringP("base", "p", "", "Override Base URL or path to use for resolving local file based or remote references")
	rootCmd.PersistentFlags().BoolP("remote", "u", true, "Allow local files and remote (http) references to be looked up")
	rootCmd.PersistentFlags().BoolP("skip-check", "k", false, "Skip checking for a valid OpenAPI document, useful for linting fragments or non-OpenAPI documents")
	rootCmd.PersistentFlags().BoolP("debug", "w", false, "Turn on debug logging")
	rootCmd.PersistentFlags().IntP("timeout", "g", 5, "Rule timeout in seconds, default is 5 seconds")
	rootCmd.PersistentFlags().BoolP("hard-mode", "z", false, "Enable all the built-in rules, even the OWASP ones. This is the level to beat!")
	rootCmd.PersistentFlags().BoolP("ext-refs", "", false, "Turn on $ref lookups and resolving for extensions (x-) objects")
	rootCmd.PersistentFlags().String("cert-file", "", "Path to client certificate file for HTTPS requests")
	rootCmd.PersistentFlags().String("key-file", "", "Path to client private key file for HTTPS requests")
	rootCmd.PersistentFlags().String("ca-file", "", "Path to CA certificate file for HTTPS requests")
	rootCmd.PersistentFlags().Bool("insecure", false, "Skip TLS certificate verification (insecure)")
	rootCmd.AddCommand(GetLintCommand())
	rootCmd.AddCommand(GetVacuumReportCommand())
	rootCmd.AddCommand(GetSpectralReportCommand())
	rootCmd.AddCommand(GetHTMLReportCommand())
	rootCmd.AddCommand(GetDashboardCommand())
	rootCmd.AddCommand(GetGenerateRulesetCommand())
	rootCmd.AddCommand(GetGenerateIgnoreFileCommand())
	rootCmd.AddCommand(GetGenerateVersionCommand())
	rootCmd.AddCommand(GetLanguageServerCommand())
	rootCmd.AddCommand(GetBundleCommand())

	if regErr := rootCmd.RegisterFlagCompletionFunc("functions", cobra.FixedCompletions(
		[]string{"so"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("ruleset", cobra.FixedCompletions(
		[]string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("timeout", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("cert-file", cobra.FixedCompletions(
		[]string{"crt", "pem", "cert"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("key-file", cobra.FixedCompletions(
		[]string{"key", "pem"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("ca-file", cobra.FixedCompletions(
		[]string{"crt", "pem", "cert"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("insecure", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}

	return rootCmd
}

func useConfigFile(cmd *cobra.Command) error {
	useEnvironmentConfiguration()
	var err error
	if len(configFile) != 0 {
		err = useUserSuppliedConfigFile(configFile)
	} else {
		err = useDefaultConfigFile()
	}
	if err != nil {
		return err
	}
	// bind global flags
	err = bindFlags(cmd.InheritedFlags(), viper.GetViper())
	if err != nil {
		return err
	}
	// bind command specific flags
	if viperSubTree := viper.Sub(cmd.Name()); viperSubTree != nil {
		err = bindFlags(cmd.LocalFlags(), viperSubTree)
	}
	return err
}

func useDefaultConfigFile() error {
	viper.SetConfigName("vacuum.conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(getXdgConfigHome())
	err := viper.ReadInConfig()
	if err == nil {
		return nil
	}
	if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		return err
	}
	// config file isn't required
	return nil
}

// Allow overriding specifying configuration from environment variables
func useEnvironmentConfiguration() {
	viper.SetEnvPrefix("VACUUM")
	viper.AutomaticEnv()
	// Environment variables can't have dashes in them
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

func useUserSuppliedConfigFile(configFilePath string) error {
	viper.SetConfigFile(os.ExpandEnv(configFilePath))
	return viper.ReadInConfig()
}

// Get config directory as per the xdg basedir spec
func getXdgConfigHome() string {
	xdgConfigHome, exists := os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		xdgConfigHome = os.Getenv("HOME") + "/.config"
	}
	return xdgConfigHome
}

// Set flag values if configuration tree has any values set
func bindFlags(flags *pflag.FlagSet, viperTree *viper.Viper) error {
	var err error
	flags.VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viperTree.IsSet(f.Name) {
			val := viperTree.Get(f.Name)
			err = flags.Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
	return err
}
