// Copyright 2025 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	wcModel "github.com/pb33f/libopenapi/what-changed/model"
)

// ChangeStats holds summary statistics about changes in a DocumentChanges
type ChangeStats struct {
	TotalChanges    int
	Added           int
	Modified        int
	Removed         int
	BreakingChanges int
}

// ExtractChangeStats analyzes DocumentChanges and returns aggregated statistics
func ExtractChangeStats(changes *wcModel.DocumentChanges) *ChangeStats {
	stats := &ChangeStats{}

	if changes == nil {
		return stats
	}

	stats.TotalChanges = changes.TotalChanges()
	stats.BreakingChanges = changes.TotalBreakingChanges()

	// Get all changes and categorize them
	if changes.PropertyChanges == nil && changes.TotalChanges() == 0 {
		return stats
	}

	allChanges := changes.GetAllChanges()
	for _, change := range allChanges {
		if change == nil {
			continue
		}

		switch change.ChangeType {
		case wcModel.Modified:
			stats.Modified++
		case wcModel.PropertyAdded, wcModel.ObjectAdded:
			stats.Added++
		case wcModel.PropertyRemoved, wcModel.ObjectRemoved:
			stats.Removed++
		}
	}

	return stats
}

// HasChanges returns true if there are any changes
func (s *ChangeStats) HasChanges() bool {
	return s.TotalChanges > 0
}
