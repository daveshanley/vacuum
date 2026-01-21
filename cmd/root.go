// Copyright 2020-2022 Dave Shanley / Quobix
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daveshanley/vacuum/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configFile  string
	versionInfo VersionInfo
	// Variables to hold ldflags values
	ldVersion string
	ldCommit  string
	ldDate    string

	// Directory that contains the active configuration file, used for resolving relative paths
	configDirectory string
)

func init() {
	// Initialize version info after Execute() is called with ldflags
}

func Execute(version, commit, date string) {
	// Store ldflags values
	ldVersion = version
	ldCommit = commit
	ldDate = date

	// Now initialize version info with ldflags available
	versionInfo = GetVersionInfo()

	if err := GetRootCommand().Execute(); err != nil {
		// Print unknown flag errors explicitly since commands have SilenceErrors: true
		// This ensures users get feedback when they mistype a flag name
		errStr := err.Error()
		if strings.Contains(errStr, "unknown flag") ||
			strings.Contains(errStr, "unknown shorthand flag") {
			tui.RenderErrorString("%s", errStr)
		}
		os.Exit(1)
	}
}

// GetVersion returns the current version string for compatibility
func GetVersion() string {
	return versionInfo.Version
}

// GetCommit returns the current commit hash for compatibility
func GetCommit() string {
	return versionInfo.Commit
}

// GetDate returns the current build date for compatibility
func GetDate() string {
	return versionInfo.Date
}

func GetRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "vacuum lint <your-openapi-file.yaml>",
		Short: "vacuum is a very fast OpenAPI linter",
		Long:  `vacuum is a very fast OpenAPI linter. It will suck all the lint off your spec in milliseconds`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := useConfigFile(cmd)
			if err != nil {
				tui.RenderError(err)
			}
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			PrintBanner()
			fmt.Println(">> Welcome! To lint something, try 'vacuum lint <my-openapi-spec.yaml>'")
			fmt.Println()
			fmt.Println("To see all the options, try 'vacuum --help'")
			fmt.Println()
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
	rootCmd.PersistentFlags().Int("lookup-timeout", 500, "Node lookup timeout in milliseconds for JSONPath queries, default is 500ms")
	rootCmd.PersistentFlags().BoolP("hard-mode", "z", false, "Enable all the built-in rules, even the OWASP ones. This is the level to beat!")
	rootCmd.PersistentFlags().BoolP("ext-refs", "", false, "Turn on $ref lookups and resolving for extensions (x-) objects")
	rootCmd.PersistentFlags().String("cert-file", "", "Path to client certificate file for HTTPS requests")
	rootCmd.PersistentFlags().String("key-file", "", "Path to client private key file for HTTPS requests")
	rootCmd.PersistentFlags().String("ca-file", "", "Path to CA certificate file for HTTPS requests")
	rootCmd.PersistentFlags().Bool("insecure", false, "Skip TLS certificate verification (insecure)")
	rootCmd.PersistentFlags().Bool("allow-private-networks", false, "Allow fetch() to access private/local networks (localhost, 10.x, 192.168.x)")
	rootCmd.PersistentFlags().Bool("allow-http", false, "Allow fetch() to use HTTP (non-HTTPS) URLs")
	rootCmd.PersistentFlags().Int("fetch-timeout", 30, "Timeout for fetch() requests in seconds (default 30)")
	rootCmd.PersistentFlags().String("changes", "", "Path to change report JSON file for filtering results to changed areas only")
	rootCmd.PersistentFlags().String("original", "", "Path to original/old spec file for inline comparison (filters results to changed areas)")
	rootCmd.PersistentFlags().Bool("changes-summary", false, "Show summary of what was filtered by --changes or --original")
	rootCmd.PersistentFlags().String("breaking-config", "", "Path to breaking rules config file (default: ./changes-rules.yaml or ~/.config/changes-rules.yaml)")
	rootCmd.PersistentFlags().Bool("warn-on-changes", false, "Inject warning violations for each detected API change")
	rootCmd.PersistentFlags().Bool("error-on-breaking", false, "Inject error violations for each breaking change")
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
	rootCmd.AddCommand(GetApplyOverlayCommand())

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
	if regErr := rootCmd.RegisterFlagCompletionFunc("lookup-timeout", cobra.NoFileCompletions); regErr != nil {
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
	if regErr := rootCmd.RegisterFlagCompletionFunc("allow-private-networks", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("allow-http", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("fetch-timeout", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("changes", cobra.FixedCompletions(
		[]string{"json"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("original", cobra.FixedCompletions(
		[]string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt,
	)); regErr != nil {
		panic(regErr)
	}
	if regErr := rootCmd.RegisterFlagCompletionFunc("changes-summary", cobra.NoFileCompletions); regErr != nil {
		panic(regErr)
	}

	return rootCmd
}

func useConfigFile(cmd *cobra.Command) error {
	configDirectory = ""
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
		setConfigDirectoryFromViper()
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
	expandedPath, err := expandUserPath(configFilePath)
	if err != nil {
		return err
	}
	viper.SetConfigFile(expandedPath)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	setConfigDirectoryFromViper()
	return nil
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

// expandUserPath expands environment variables and a leading ~ in a user-supplied path.
func expandUserPath(pathValue string) (string, error) {
	if pathValue == "" {
		return "", nil
	}

	expanded := os.ExpandEnv(pathValue)

	if strings.HasPrefix(expanded, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to resolve home directory: %w", err)
		}
		if expanded == "~" {
			expanded = home
		} else if strings.HasPrefix(expanded, "~/") || strings.HasPrefix(expanded, "~\\") {
			expanded = filepath.Join(home, expanded[2:])
		}
	}

	return expanded, nil
}

// setConfigDirectoryFromViper captures the directory of the currently loaded configuration file, if any.
func setConfigDirectoryFromViper() {
	if used := viper.ConfigFileUsed(); used != "" {
		if absPath, err := filepath.Abs(used); err == nil {
			configDirectory = filepath.Dir(absPath)
		} else {
			configDirectory = filepath.Dir(used)
		}
	}
}

// ResolveConfigPath normalizes paths supplied via flags or configuration.
// It expands ~ and environment variables. For relative paths, it checks
// CWD first, then falls back to the config directory if the path doesn't exist.
func ResolveConfigPath(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}

	// Skip resolution for URLs or other schemes
	if strings.Contains(raw, "://") {
		return raw, nil
	}

	expanded, err := expandUserPath(raw)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to resolve working directory: %w", err)
	}

	// Check CWD first for relative paths
	cwdPath := filepath.Clean(filepath.Join(cwd, expanded))
	if _, err := os.Stat(cwdPath); err == nil {
		return cwdPath, nil
	}

	// Fall back to config directory if path doesn't exist in CWD
	if configDirectory != "" {
		configPath := filepath.Clean(filepath.Join(configDirectory, expanded))
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// Default to CWD path (will fail later with appropriate error)
	return cwdPath, nil
}
