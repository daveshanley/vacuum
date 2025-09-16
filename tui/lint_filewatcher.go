// Copyright 2023-2025 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

package tui

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/daveshanley/vacuum/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/pb33f/libopenapi/index"
	"go.yaml.in/yaml/v4"
)

// WatchConfig holds configuration for file watching
type WatchConfig struct {
	Enabled         bool
	BaseFlag        string
	SkipCheckFlag   bool
	TimeoutFlag     int
	HardModeFlag    bool
	RemoteFlag      bool
	IgnoreFile      string
	FunctionsFlag   string
	RulesetFlag     string
	CertFile        string
	KeyFile         string
	CAFile          string
	Insecure        bool
	Silent          bool
	CustomFunctions map[string]model.RuleFunction // Pre-loaded custom functions
}

// setupFileWatcher initializes file watching if enabled
func (m *ViolationResultTableModel) setupFileWatcher() tea.Cmd {
	if m.watchConfig == nil || !m.watchConfig.Enabled {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to create file watcher: %v", err)
		m.watchState = WatchStateError
		return nil
	}

	m.watcher = watcher

	absPath, err := filepath.Abs(m.fileName)
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to get absolute path for %s: %v", m.fileName, err)
		m.watchState = WatchStateError
		return nil
	}

	err = m.watcher.Add(absPath)
	if err != nil {
		m.watchError = fmt.Sprintf("Failed to watch file %s: %v", absPath, err)
		m.watchState = WatchStateError
		return nil
	}

	m.watchedFiles = []string{absPath}

	go m.watchFileChanges()

	return m.listenForChannelMessages()
}

// watchFileChanges runs in a goroutine to monitor file system events
func (m *ViolationResultTableModel) watchFileChanges() {
	for {
		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				select {
				case m.watchMsgChan <- fileChangeMsg{fileName: event.Name}:
				default:
				}
			}

		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			select {
			case m.watchMsgChan <- relintErrorMsg{err: fmt.Errorf("file watcher error: %w", err)}:
			default:
			}
		}
	}
}

// listenForChannelMessages returns a command that listens for messages from the watcher channel
func (m *ViolationResultTableModel) listenForChannelMessages() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		select {
		case msg := <-m.watchMsgChan:
			return msg
		case <-time.After(100 * time.Millisecond):
			return continueWatchingMsg{}
		}
	})
}

// handleFileChange processes file change events with debouncing
func (m *ViolationResultTableModel) handleFileChange(fileName string) tea.Cmd {
	m.lastChangeTime = time.Now()

	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}

	// new debounce timer
	m.debounceTimer = time.NewTimer(WatchDebounceDelay)

	return tea.Cmd(func() tea.Msg {
		<-m.debounceTimer.C

		if time.Since(m.lastChangeTime) >= WatchDebounceDelay {
			return m.performRelint()
		}

		return nil
	})
}

// performRelint re-lints the specification with current configuration
func (m *ViolationResultTableModel) performRelint() tea.Msg {
	m.watchState = WatchStateProcessing

	currentRow := m.table.Cursor()

	specBytes, err := os.ReadFile(m.fileName)
	if err != nil {
		return relintErrorMsg{err: fmt.Errorf("failed to read spec file: %w", err)}
	}

	// Restored working linting logic
	var bufferedLogger *slog.Logger
	bufferedLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	defaultRuleSets := rulesets.BuildDefaultRuleSetsWithLogger(bufferedLogger)
	selectedRS := defaultRuleSets.GenerateOpenAPIRecommendedRuleSet()

	// Use pre-loaded custom functions from dashboard command
	customFuncs := m.watchConfig.CustomFunctions

	// hard mode
	if m.watchConfig.HardModeFlag {
		selectedRS = defaultRuleSets.GenerateOpenAPIDefaultRuleSet()
		owaspRules := rulesets.GetAllOWASPRules()
		for k, v := range owaspRules {
			selectedRS.Rules[k] = v
		}
	}

	// custom ruleset if specified
	if m.watchConfig.RulesetFlag != "" {
		if strings.HasPrefix(m.watchConfig.RulesetFlag, "http") {
			if !m.watchConfig.RemoteFlag {
				return relintErrorMsg{err: fmt.Errorf("remote ruleset specified but remote flag is disabled")}
			}
			downloadedRS, rsErr := rulesets.DownloadRemoteRuleSet(context.Background(), m.watchConfig.RulesetFlag, nil)
			if rsErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to load remote ruleset '%s': %w", m.watchConfig.RulesetFlag, rsErr)}
			}
			selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(downloadedRS)
		} else {
			rsBytes, rsErr := os.ReadFile(m.watchConfig.RulesetFlag)
			if rsErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to read ruleset file '%s': %w", m.watchConfig.RulesetFlag, rsErr)}
			}
			userRS, userErr := rulesets.CreateRuleSetFromData(rsBytes)
			if userErr != nil {
				return relintErrorMsg{err: fmt.Errorf("unable to parse ruleset file '%s': %w", m.watchConfig.RulesetFlag, userErr)}
			}
			selectedRS = defaultRuleSets.GenerateRuleSetFromSuppliedRuleSet(userRS)
		}

		// Merge OWASP rules if hard mode is enabled
		if m.watchConfig.HardModeFlag {
			owaspRules := rulesets.GetAllOWASPRules()
			if selectedRS.Rules == nil {
				selectedRS.Rules = make(map[string]*model.Rule)
			}
			for k, v := range owaspRules {
				if selectedRS.Rules[k] == nil {
					selectedRS.Rules[k] = v
				}
			}
		}
	}

	// ignore file if specified
	var ignoredItems model.IgnoredItems
	if m.watchConfig.IgnoreFile != "" {
		raw, ferr := os.ReadFile(m.watchConfig.IgnoreFile)
		if ferr == nil {
			_ = yaml.Unmarshal(raw, &ignoredItems)
		}
	}

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           selectedRS,
		Spec:              specBytes,
		SpecFileName:      m.fileName,
		CustomFunctions:   customFuncs,
		Base:              m.watchConfig.BaseFlag,
		AllowLookup:       m.watchConfig.RemoteFlag,
		SkipDocumentCheck: m.watchConfig.SkipCheckFlag,
		Logger:            bufferedLogger,
		Timeout:           time.Duration(m.watchConfig.TimeoutFlag) * time.Second,
		HTTPClientConfig: utils.HTTPClientConfig{
			CertFile: m.watchConfig.CertFile,
			KeyFile:  m.watchConfig.KeyFile,
			CAFile:   m.watchConfig.CAFile,
			Insecure: m.watchConfig.Insecure,
		},
	})

	m.updateWatchedFilesFromRolodex(result.Index)

	if len(result.Errors) > 0 {
		return relintErrorMsg{err: fmt.Errorf("linting failed: %v", result.Errors[0])}
	}

	filteredResults := utils.FilterIgnoredResults(result.Results, ignoredItems)

	// Create result set and sort by line number
	tempResultSet := model.NewRuleResultSet(filteredResults)
	tempResultSet.SortResultsByLineNumber()
	sortedResults := tempResultSet.Results

	resultPointers := make([]*model.RuleFunctionResult, len(sortedResults))
	for i := range sortedResults {
		resultPointers[i] = sortedResults[i]
	}

	return relintCompleteMsg{
		results:     resultPointers,
		specContent: specBytes,
		selectedRow: currentRow,
	}
}

// updateWatchedFilesFromRolodex adds all files from the document rolodex to the watcher
func (m *ViolationResultTableModel) updateWatchedFilesFromRolodex(specIndex *index.SpecIndex) {
	if m.watcher == nil || specIndex == nil {
		return
	}

	rolodex := specIndex.GetRolodex()
	if rolodex == nil {
		return
	}

	// track new files to avoid duplicates
	newFiles := make(map[string]bool)
	for _, existingFile := range m.watchedFiles {
		newFiles[existingFile] = true
	}

	allIndexes := rolodex.GetIndexes()
	for _, idx := range allIndexes {
		if idx == nil {
			continue
		}

		config := idx.GetConfig()
		if config != nil && config.SpecFilePath != "" {
			filePath := config.SpecFilePath

			absPath, err := filepath.Abs(filePath)
			if err != nil {
				continue
			}

			// skip if already watching
			if newFiles[absPath] {
				continue
			}

			// add to watcher
			err = m.watcher.Add(absPath)
			if err != nil {
				continue
			}

			m.watchedFiles = append(m.watchedFiles, absPath)
			newFiles[absPath] = true
		}
	}
}
