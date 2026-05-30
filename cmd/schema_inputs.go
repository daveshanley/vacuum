// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func defaultSchemaFolderIncludes() []string {
	return []string{"**/*.json", "**/*.yaml", "**/*.yml"}
}

func collectSchemaInputs(cmd *cobra.Command, args, globPatterns, includes, excludes []string, stdin bool, baseFlag, mode string) ([]schemaInput, error) {
	if stdin {
		if len(args) > 0 || len(globPatterns) > 0 {
			return nil, fmt.Errorf("schema %s --stdin cannot be combined with files, folders, or --globbed-files", mode)
		}
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(cmd.InOrStdin()); err != nil {
			return nil, err
		}
		base := baseFlag
		if base == "" {
			base = "."
		}
		resolvedBase, err := filepath.Abs(base)
		if err != nil {
			return nil, err
		}
		return []schemaInput{{
			Path:      "stdin",
			Display:   "stdin",
			Bytes:     buf.Bytes(),
			Base:      resolvedBase,
			FromStdin: true,
		}}, nil
	}

	var files []string
	var folderInputs []string
	var fileInputs []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			folderInputs = append(folderInputs, arg)
			if len(includes) == 0 {
				includes = defaultSchemaFolderIncludes()
			}
			rootFiles, walkErr := schemaFilesFromFolder(arg, includes, excludes)
			if walkErr != nil {
				return nil, walkErr
			}
			files = append(files, rootFiles...)
			continue
		}
		fileInputs = append(fileInputs, arg)
		files = append(files, arg)
	}
	if len(folderInputs) > 0 && len(fileInputs) > 0 {
		return nil, fmt.Errorf("folder inputs cannot be combined with file inputs (%s); if you used --include with a shell glob, quote it, for example --include \"**/*.json\"", strings.Join(fileInputs, ", "))
	}
	globFiles, err := schemaFilesFromGlobs(globPatterns)
	if err != nil {
		return nil, err
	}
	files = append(files, globFiles...)
	files = deduplicate(files)
	sort.Strings(files)

	if len(files) == 0 {
		return nil, nil
	}
	inputs := make([]schemaInput, 0, len(files))
	for _, file := range files {
		raw, readErr := os.ReadFile(file)
		if readErr != nil {
			return nil, readErr
		}
		base, baseErr := ResolveBasePathForFile(file, baseFlag)
		if baseErr != nil {
			return nil, baseErr
		}
		inputs = append(inputs, schemaInput{
			Path:    file,
			Display: file,
			Bytes:   raw,
			Base:    base,
		})
	}
	return inputs, nil
}

func schemaFilesFromFolder(root string, includes, excludes []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() {
			if strings.HasPrefix(name, ".") && file != root {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !globListMatches(includes, rel) || globListMatches(excludes, rel) {
			return nil
		}
		files = append(files, file)
		return nil
	})
	return files, err
}

func schemaFilesFromGlobs(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		for _, expanded := range expandBracePattern(pattern) {
			matches, err := schemaGlob(expanded)
			if err != nil {
				return nil, err
			}
			files = append(files, matches...)
		}
	}
	return deduplicate(files), nil
}

func schemaGlob(pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		return schemaRegularFiles(matches), nil
	}
	root := globWalkRoot(pattern)
	re, err := globRegex(filepath.ToSlash(pattern))
	if err != nil {
		return nil, err
	}
	var matches []string
	err = filepath.WalkDir(root, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		slashFile := filepath.ToSlash(file)
		if re.MatchString(slashFile) {
			matches = append(matches, file)
		}
		return nil
	})
	return matches, err
}

func schemaRegularFiles(paths []string) []string {
	var files []string
	for _, candidate := range paths {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			files = append(files, candidate)
		}
	}
	return files
}

func globWalkRoot(pattern string) string {
	clean := filepath.Clean(pattern)
	absolute := filepath.IsAbs(clean)
	if absolute {
		clean = strings.TrimPrefix(clean, string(filepath.Separator))
	}
	parts := strings.Split(clean, string(filepath.Separator))
	var rootParts []string
	for _, part := range parts {
		if strings.ContainsAny(part, "*?[") {
			break
		}
		rootParts = append(rootParts, part)
	}
	if len(rootParts) == 0 {
		if absolute {
			return string(filepath.Separator)
		}
		return "."
	}
	root := filepath.Join(rootParts...)
	if absolute {
		root = string(filepath.Separator) + root
	}
	if root == "" {
		return "."
	}
	return root
}

func globListMatches(patterns []string, candidate string) bool {
	for _, pattern := range patterns {
		for _, expanded := range expandBracePattern(filepath.ToSlash(pattern)) {
			re, err := globRegex(expanded)
			if err == nil && re.MatchString(candidate) {
				return true
			}
		}
	}
	return false
}

func globRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 2
				} else {
					b.WriteString(".*")
					i++
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}
