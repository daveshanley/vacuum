// Package utils provides utility functions for vacuum
package utils

import (
	"fmt"
	"strings"
)

// RenderTextTable creates a simple text table from 2D string slice data
func RenderTextTable(data [][]string, hasHeader bool) error {
	if len(data) == 0 {
		return nil
	}
	
	// Calculate column widths
	colWidths := make([]int, len(data[0]))
	for _, row := range data {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	
	// Print header
	if hasHeader && len(data) > 0 {
		for i, cell := range data[0] {
			fmt.Printf("%-*s", colWidths[i]+2, cell)
		}
		fmt.Println()
		
		// Print separator
		for _, width := range colWidths {
			fmt.Print(strings.Repeat("-", width+2))
		}
		fmt.Println()
	}
	
	// Print data rows
	startRow := 0
	if hasHeader {
		startRow = 1
	}
	
	for i := startRow; i < len(data); i++ {
		for j, cell := range data[i] {
			if j < len(colWidths) {
				fmt.Printf("%-*s", colWidths[j]+2, cell)
			}
		}
		fmt.Println()
	}
	
	return nil
}

// RenderHeader creates a simple text header
func RenderHeader(text string) {
	width := 80 // Default width
	
	fmt.Println()
	fmt.Println(strings.Repeat("=", width))
	fmt.Println(text)
	fmt.Println(strings.Repeat("=", width))
	fmt.Println()
}

// RenderBox creates a simple text box around the given text
func RenderBox(text string) {
	lines := strings.Split(text, "\n")
	width := 0
	
	// Find the maximum line width
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	
	// Add padding
	width += 4
	
	// Top border
	fmt.Println("+" + strings.Repeat("-", width) + "+")
	
	// Content
	for _, line := range lines {
		fmt.Printf("| %-*s |\n", width-2, line)
	}
	
	// Bottom border
	fmt.Println("+" + strings.Repeat("-", width) + "+")
}