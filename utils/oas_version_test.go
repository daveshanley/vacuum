// Copyright 2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/stretchr/testify/assert"
)

func TestIsOAS30(t *testing.T) {
	tests := []struct {
		name string
		info *datamodel.SpecInfo
		want bool
	}{
		{"nil spec info", nil, false},
		{"OAS 3.0.0", &datamodel.SpecInfo{VersionNumeric: 3.0}, true},
		{"OAS 3.0.3", &datamodel.SpecInfo{VersionNumeric: 3.03}, true},
		{"OAS 3.1.0", &datamodel.SpecInfo{VersionNumeric: 3.1}, false},
		{"OAS 3.2.0", &datamodel.SpecInfo{VersionNumeric: 3.2}, false},
		{"OAS 2.0", &datamodel.SpecInfo{VersionNumeric: 2.0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsOAS30(tt.info))
		})
	}
}
