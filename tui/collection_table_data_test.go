// Copyright 2026 Princess Beef Heavy Industries, LLC / Dave Shanley
// https://pb33f.io

// SPDX-License-Identifier: MIT

package tui

import (
	"testing"

	"github.com/pb33f/doctor/frank"
	"github.com/stretchr/testify/assert"
)

func TestBuildCollectionTable_NilResult(t *testing.T) {
	columns, rows := BuildCollectionTable(nil, 120)
	assert.Nil(t, columns)
	assert.Nil(t, rows)
}

func TestBuildCollectionTable_EmptyResult(t *testing.T) {
	result := &frank.FrankResult{}
	columns, rows := BuildCollectionTable(result, 120)
	assert.NotNil(t, columns)
	assert.Len(t, columns, 4)
	assert.Empty(t, rows)
}

func TestBuildCollectionTable_SingleFolderSingleRequest(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Folder: &frank.Folder{
					Info: frank.FolderInfo{Name: "Users"},
				},
				Requests: []*frank.Request{
					{
						Info: frank.RequestInfo{Name: "List Users"},
						HTTP: frank.RequestHTTP{
							Method: "GET",
							URL:    "{{baseUrl}}/users",
						},
					},
				},
			},
		},
	}

	columns, rows := BuildCollectionTable(result, 120)
	assert.Len(t, columns, 4)
	assert.Len(t, rows, 1)
	assert.Equal(t, "Users", rows[0][0])
	assert.Equal(t, "GET", rows[0][1])
	assert.Equal(t, "{{baseUrl}}/users", rows[0][2])
	assert.Equal(t, "List Users", rows[0][3])
}

func TestBuildCollectionTable_MultipleFoldersMultipleRequests(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Folder: &frank.Folder{
					Info: frank.FolderInfo{Name: "Users"},
				},
				Requests: []*frank.Request{
					{
						Info: frank.RequestInfo{Name: "List Users"},
						HTTP: frank.RequestHTTP{Method: "GET", URL: "{{baseUrl}}/users"},
					},
					{
						Info: frank.RequestInfo{Name: "Create User"},
						HTTP: frank.RequestHTTP{Method: "POST", URL: "{{baseUrl}}/users"},
					},
				},
			},
			{
				Folder: &frank.Folder{
					Info: frank.FolderInfo{Name: "Products"},
				},
				Requests: []*frank.Request{
					{
						Info: frank.RequestInfo{Name: "Get Product"},
						HTTP: frank.RequestHTTP{Method: "GET", URL: "{{baseUrl}}/products/{id}"},
					},
				},
			},
		},
	}

	columns, rows := BuildCollectionTable(result, 120)
	assert.Len(t, columns, 4)
	assert.Len(t, rows, 3)

	assert.Equal(t, "Users", rows[0][0])
	assert.Equal(t, "GET", rows[0][1])
	assert.Equal(t, "Users", rows[1][0])
	assert.Equal(t, "POST", rows[1][1])
	assert.Equal(t, "Products", rows[2][0])
	assert.Equal(t, "GET", rows[2][1])
}

func TestBuildCollectionTable_ColumnWidthsRespond(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Folder: &frank.Folder{
					Info: frank.FolderInfo{Name: "Users"},
				},
				Requests: []*frank.Request{
					{
						Info: frank.RequestInfo{Name: "List Users"},
						HTTP: frank.RequestHTTP{Method: "GET", URL: "{{baseUrl}}/users"},
					},
				},
			},
		},
	}

	// Wide terminal
	columnsWide, _ := BuildCollectionTable(result, 200)
	// Narrow terminal
	columnsNarrow, _ := BuildCollectionTable(result, 60)

	// URL column (index 2) should be wider on a wider terminal
	assert.Greater(t, columnsWide[2].Width, columnsNarrow[2].Width)
}

func TestFormatCollectionSummary_WithEnvironments(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Requests: []*frank.Request{
					{HTTP: frank.RequestHTTP{Method: "GET"}},
					{HTTP: frank.RequestHTTP{Method: "POST"}},
				},
			},
		},
		Environments: []*frank.Environment{
			{Name: "dev"},
			{Name: "prod"},
		},
	}

	summary := FormatCollectionSummary(result)
	assert.Equal(t, "2 requests across 1 folders (2 environments generated)", summary)
}

func TestFormatCollectionSummary_WithoutEnvironments(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Requests: []*frank.Request{
					{HTTP: frank.RequestHTTP{Method: "GET"}},
				},
			},
			{
				Requests: []*frank.Request{
					{HTTP: frank.RequestHTTP{Method: "DELETE"}},
				},
			},
		},
	}

	summary := FormatCollectionSummary(result)
	assert.Equal(t, "2 requests across 2 folders", summary)
}

func TestFormatCollectionSummary_Nil(t *testing.T) {
	summary := FormatCollectionSummary(nil)
	assert.Equal(t, "0 requests across 0 folders", summary)
}

func TestCountCollectionRequests(t *testing.T) {
	result := &frank.FrankResult{
		Folders: []*frank.FolderOutput{
			{
				Requests: []*frank.Request{
					{HTTP: frank.RequestHTTP{Method: "GET"}},
					{HTTP: frank.RequestHTTP{Method: "POST"}},
				},
			},
			{
				Requests: []*frank.Request{
					{HTTP: frank.RequestHTTP{Method: "PUT"}},
				},
			},
		},
	}

	assert.Equal(t, 3, CountCollectionRequests(result))
}

func TestCountCollectionRequests_Nil(t *testing.T) {
	assert.Equal(t, 0, CountCollectionRequests(nil))
}
