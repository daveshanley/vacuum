// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package rulesets

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/daveshanley/vacuum/model"
)

type bufferedRuleSetLogRecord struct {
	level   slog.Level
	message string
	attrs   []slog.Attr
}

type bufferedRuleSetLogStore struct {
	mutex   sync.Mutex
	records []bufferedRuleSetLogRecord
}

type bufferedRuleSetLogHandler struct {
	store *bufferedRuleSetLogStore
	attrs []slog.Attr
}

func newBufferedRuleSetLogger() (*slog.Logger, *bufferedRuleSetLogStore) {
	store := &bufferedRuleSetLogStore{}
	return slog.New(&bufferedRuleSetLogHandler{store: store}), store
}

func (h *bufferedRuleSetLogHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *bufferedRuleSetLogHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := append([]slog.Attr(nil), h.attrs...)
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	h.store.mutex.Lock()
	h.store.records = append(h.store.records, bufferedRuleSetLogRecord{
		level:   record.Level,
		message: record.Message,
		attrs:   attrs,
	})
	h.store.mutex.Unlock()
	return nil
}

func (h *bufferedRuleSetLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &bufferedRuleSetLogHandler{
		store: h.store,
		attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...),
	}
}

func (h *bufferedRuleSetLogHandler) WithGroup(_ string) slog.Handler {
	return h
}

func (s *bufferedRuleSetLogStore) snapshot() []bufferedRuleSetLogRecord {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	records := make([]bufferedRuleSetLogRecord, len(s.records))
	for i, record := range s.records {
		records[i] = bufferedRuleSetLogRecord{
			level:   record.level,
			message: record.message,
			attrs:   append([]slog.Attr(nil), record.attrs...),
		}
	}
	return records
}

func flushBufferedRuleSetLogs(logger *slog.Logger, records []bufferedRuleSetLogRecord) {
	for _, record := range records {
		logger.LogAttrs(context.Background(), record.level, record.message, record.attrs...)
	}
}

func (rsm ruleSetsModel) loadExternalRulesetsWithTimeout(extends map[string]string, rs *RuleSet, httpClient *http.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), externalRulesetFetchTimeout)
	defer cancel()

	for location := range extends {
		if !isExternalRulesetLocation(location) {
			continue
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			rsm.logger.Error("external ruleset fetch timed out", "timeout", externalRulesetFetchTimeout)
			break
		}

		remote := strings.HasPrefix(location, "http")
		if !rsm.loadExternalRulesetWithTimeout(ctx, location, rs, remote, httpClient) {
			rsm.logger.Error("external ruleset fetch timed out", "timeout", externalRulesetFetchTimeout)
			break
		}
	}
}

func (rsm ruleSetsModel) loadExternalRulesetWithTimeout(ctx context.Context, location string, rs *RuleSet, remote bool, httpClient *http.Client) bool {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return false
	}

	workingRuleSet := cloneRuleSetForExternalLoad(rs)
	workerLogger, bufferedLogs := newBufferedRuleSetLogger()
	workerRuleSets := rsm
	workerRuleSets.logger = workerLogger

	done := make(chan struct{})
	go func() {
		defer close(done)
		SniffOutAllExternalRules(ctx, &workerRuleSets, location, nil, workingRuleSet, remote, httpClient)
	}()

	select {
	case <-done:
		copyRuleSetExternalState(rs, workingRuleSet)
		flushBufferedRuleSetLogs(rsm.logger, bufferedLogs.snapshot())
		return !errors.Is(ctx.Err(), context.DeadlineExceeded)
	case <-ctx.Done():
		copyRuleSetExternalState(rs, workingRuleSet)
		flushBufferedRuleSetLogs(rsm.logger, bufferedLogs.snapshot())
		return false
	}
}

func isExternalRulesetLocation(location string) bool {
	return strings.HasPrefix(location, "http") ||
		filepath.Ext(location) == ".yml" ||
		filepath.Ext(location) == ".yaml" ||
		filepath.Ext(location) == ".json"
}

func cloneRuleSetForExternalLoad(source *RuleSet) *RuleSet {
	if source == nil {
		return &RuleSet{}
	}

	source.mutex.Lock()
	defer source.mutex.Unlock()

	return &RuleSet{
		Description:      source.Description,
		DocumentationURI: source.DocumentationURI,
		Formats:          append([]string(nil), source.Formats...),
		RuleDefinitions:  cloneRuleDefinitionMap(source.RuleDefinitions),
		Rules:            cloneRuleMap(source.Rules),
		Extends:          source.Extends,
		Aliases:          cloneInterfaceMap(source.Aliases),
		ParsedAliases:    cloneParsedAliasMap(source.ParsedAliases),
		extendsMeta:      cloneStringMap(source.extendsMeta),
	}
}

func copyRuleSetExternalState(target, source *RuleSet) {
	if target == nil || source == nil || target == source {
		return
	}

	source.mutex.Lock()
	defer source.mutex.Unlock()
	target.mutex.Lock()
	defer target.mutex.Unlock()

	target.Description = source.Description
	target.DocumentationURI = source.DocumentationURI
	target.Formats = append([]string(nil), source.Formats...)
	target.RuleDefinitions = replaceRuleDefinitionMap(target.RuleDefinitions, source.RuleDefinitions)
	target.Rules = replaceRuleMap(target.Rules, source.Rules)
	target.Aliases = replaceInterfaceMap(target.Aliases, source.Aliases)
	target.ParsedAliases = replaceParsedAliasMap(target.ParsedAliases, source.ParsedAliases)
	target.extendsMeta = replaceStringMap(target.extendsMeta, source.extendsMeta)
}

func cloneRuleDefinitionMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = cloneRuleDefinition(value)
	}
	return cloned
}

func cloneRuleMap(source map[string]*model.Rule) map[string]*model.Rule {
	if source == nil {
		return nil
	}
	cloned := make(map[string]*model.Rule, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneInterfaceMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneParsedAliasMap(source map[string]*ParsedAlias) map[string]*ParsedAlias {
	if source == nil {
		return nil
	}
	cloned := make(map[string]*ParsedAlias, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func replaceRuleDefinitionMap(target, source map[string]interface{}) map[string]interface{} {
	if target == nil {
		return cloneRuleDefinitionMap(source)
	}
	clear(target)
	for key, value := range source {
		target[key] = cloneRuleDefinition(value)
	}
	return target
}

func replaceRuleMap(target, source map[string]*model.Rule) map[string]*model.Rule {
	if target == nil {
		return cloneRuleMap(source)
	}
	clear(target)
	for key, value := range source {
		target[key] = value
	}
	return target
}

func replaceInterfaceMap(target, source map[string]interface{}) map[string]interface{} {
	if target == nil {
		return cloneInterfaceMap(source)
	}
	clear(target)
	for key, value := range source {
		target[key] = value
	}
	return target
}

func replaceParsedAliasMap(target, source map[string]*ParsedAlias) map[string]*ParsedAlias {
	if target == nil {
		return cloneParsedAliasMap(source)
	}
	clear(target)
	for key, value := range source {
		target[key] = value
	}
	return target
}

func replaceStringMap(target, source map[string]string) map[string]string {
	if target == nil {
		return cloneStringMap(source)
	}
	clear(target)
	for key, value := range source {
		target[key] = value
	}
	return target
}
