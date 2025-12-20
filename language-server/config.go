// Copyright 2024-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package languageserver

import (
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// initializeConfig loads the vacuum configuration on language server startup
func (s *ServerState) initializeConfig() {
	// Set up environment variable support
	viper.SetEnvPrefix("VACUUM")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Try to load config file
	viper.SetConfigName("vacuum.conf")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath(getXdgConfigHome())

	// Read the config file (ignore if not found)
	_ = viper.ReadInConfig()

	// Parse file config and apply it
	s.fileConfig = s.parseViperConfig()
	if err := s.applyEffectiveConfig(); err != nil {
		s.logger.Warn("failed to apply initial config", "error", err)
	}
}

// getXdgConfigHome gets config directory as per the xdg basedir spec
func getXdgConfigHome() string {
	xdgConfigHome, exists := os.LookupEnv("XDG_CONFIG_HOME")
	if !exists {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		xdgConfigHome = home + "/.config"
	}
	return xdgConfigHome + "/vacuum"
}

// parseViperConfig extracts configuration from viper into an LSPConfig struct
func (s *ServerState) parseViperConfig() *LSPConfig {
	cfg := &LSPConfig{}

	if viper.IsSet("ruleset") {
		cfg.Ruleset = viper.GetString("ruleset")
	}
	if viper.IsSet("functions") {
		cfg.Functions = viper.GetString("functions")
	}
	if viper.IsSet("base") {
		cfg.Base = viper.GetString("base")
	}
	if viper.IsSet("remote") {
		cfg.Remote = boolPtr(viper.GetBool("remote"))
	}
	if viper.IsSet("skip-check") {
		cfg.SkipCheck = boolPtr(viper.GetBool("skip-check"))
	}
	if viper.IsSet("timeout") {
		cfg.Timeout = intPtr(viper.GetInt("timeout"))
	}
	if viper.IsSet("lookup-timeout") {
		cfg.LookupTimeout = intPtr(viper.GetInt("lookup-timeout"))
	}
	if viper.IsSet("hard-mode") {
		cfg.HardMode = boolPtr(viper.GetBool("hard-mode"))
	}
	if viper.IsSet("ignore-array-circle-ref") {
		cfg.IgnoreArrayCircleRef = boolPtr(viper.GetBool("ignore-array-circle-ref"))
	}
	if viper.IsSet("ignore-polymorph-circle-ref") {
		cfg.IgnorePolymorphCircleRef = boolPtr(viper.GetBool("ignore-polymorph-circle-ref"))
	}
	if viper.IsSet("ext-refs") {
		cfg.ExtensionRefs = boolPtr(viper.GetBool("ext-refs"))
	}
	if viper.IsSet("ignore-file") {
		cfg.IgnoreFile = viper.GetString("ignore-file")
	}
	if viper.IsSet("cert-file") {
		cfg.CertFile = viper.GetString("cert-file")
	}
	if viper.IsSet("key-file") {
		cfg.KeyFile = viper.GetString("key-file")
	}
	if viper.IsSet("ca-file") {
		cfg.CAFile = viper.GetString("ca-file")
	}
	if viper.IsSet("insecure") {
		cfg.Insecure = boolPtr(viper.GetBool("insecure"))
	}

	return cfg
}

// onConfigChange is called when the config file changes on disk
func (s *ServerState) onConfigChange(e fsnotify.Event) {
	s.logger.Info("config file changed, reloading", "file", e.Name)

	// Re-parse the config from viper
	s.fileConfig = s.parseViperConfig()

	// Apply the updated configuration
	if err := s.applyEffectiveConfig(); err != nil {
		s.logger.Warn("failed to apply config change", "error", err)
		return
	}

	// Re-lint all open documents if we have a notify function
	if notify := s.getNotifyFunc(); notify != nil {
		s.relintAllDocuments(notify)
	}
}
