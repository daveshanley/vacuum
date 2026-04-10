// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// SPDX-License-Identifier: MIT

package tui

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"github.com/pb33f/doctor/frank"
)

// collectionColumnWidths holds the calculated widths for each column
type collectionColumnWidths struct {
	folder int
	method int
	url    int
	name   int
}

// BuildCollectionTable builds the collection table with responsive column widths
func BuildCollectionTable(result *frank.FrankResult, terminalWidth int) ([]table.Column, []table.Row) {
	if result == nil {
		return nil, nil
	}

	rows := buildCollectionTableRows(result)
	widths := calculateCollectionColumnWidths(result, terminalWidth)
	columns := buildCollectionTableColumns(widths)

	return columns, rows
}

// buildCollectionTableRows creates one row per request across all folders
func buildCollectionTableRows(result *frank.FrankResult) []table.Row {
	var rows []table.Row

	for _, folderOut := range result.Folders {
		folderName := ""
		if folderOut.Folder != nil {
			folderName = folderOut.Folder.Info.Name
		}

		for _, req := range folderOut.Requests {
			method := ""
			url := ""
			name := ""

			if req != nil {
				method = req.HTTP.Method
				url = req.HTTP.URL
				name = req.Info.Name
			}

			rows = append(rows, table.Row{
				folderName,
				method,
				url,
				name,
			})
		}
	}

	return rows
}

// calculateCollectionColumnWidths calculates responsive column widths based on terminal size.
// The strategy: folder is sized to content (capped), method is fixed, and the remaining
// space is split between URL and Name at a 55/45 ratio so names stay readable.
func calculateCollectionColumnWidths(result *frank.FrankResult, terminalWidth int) collectionColumnWidths {
	const (
		methodWidth    = 8  // Fixed: "DELETE" is the longest at 6, plus padding
		maxFolderCap   = 20 // Folder names are short — don't let them bloat
		minFolderWidth = 10
		minURLWidth    = 20
		minNameWidth   = 20
	)

	columnCount := 4
	columnPadding := columnCount * 2
	availableWidth := terminalWidth - 2 - columnPadding

	// Measure actual folder content width
	contentFolderWidth := len("Folder")
	for _, folderOut := range result.Folders {
		if folderOut.Folder != nil && len(folderOut.Folder.Info.Name) > contentFolderWidth {
			contentFolderWidth = len(folderOut.Folder.Info.Name)
		}
	}
	if contentFolderWidth > maxFolderCap {
		contentFolderWidth = maxFolderCap
	}

	folderWidth := contentFolderWidth

	// Remaining space after fixed columns goes to URL + Name
	remaining := availableWidth - methodWidth - folderWidth
	if remaining < minURLWidth+minNameWidth {
		remaining = minURLWidth + minNameWidth
	}

	// Split remaining 55% URL, 45% Name
	urlWidth := remaining * 55 / 100
	nameWidth := remaining - urlWidth

	// Enforce minimums
	if urlWidth < minURLWidth {
		urlWidth = minURLWidth
	}
	if nameWidth < minNameWidth {
		nameWidth = minNameWidth
	}
	if folderWidth < minFolderWidth {
		folderWidth = minFolderWidth
	}

	return collectionColumnWidths{
		folder: folderWidth,
		method: methodWidth,
		url:    urlWidth,
		name:   nameWidth,
	}
}

// buildCollectionTableColumns creates the table column definitions
func buildCollectionTableColumns(widths collectionColumnWidths) []table.Column {
	return []table.Column{
		{Title: "Folder", Width: widths.folder},
		{Title: "Method", Width: widths.method},
		{Title: "URL", Width: widths.url},
		{Title: "Name", Width: widths.name},
	}
}

// FormatCollectionSummary formats the summary line for collection generation
func FormatCollectionSummary(result *frank.FrankResult) string {
	if result == nil {
		return "0 requests across 0 folders"
	}

	totalRequests := CountCollectionRequests(result)
	folderCount := len(result.Folders)
	envCount := len(result.Environments)

	if envCount > 0 {
		return fmt.Sprintf("%d requests across %d folders (%d environments generated)", totalRequests, folderCount, envCount)
	}
	return fmt.Sprintf("%d requests across %d folders", totalRequests, folderCount)
}

// CountCollectionRequests totals requests across all folders
func CountCollectionRequests(result *frank.FrankResult) int {
	if result == nil {
		return 0
	}

	total := 0
	for _, folderOut := range result.Folders {
		total += len(folderOut.Requests)
	}
	return total
}
