// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

package utils

import (
	"testing"

	"github.com/pb33f/testify/assert"
)

func TestBuildStablePrimaryAndPaths_CanonicalFirstSortedDeduped(t *testing.T) {
	canonical := "$.components.schemas['Pet']"
	primaryPath, allPaths := buildStablePrimaryAndPaths(canonical, []string{
		"$.paths['/pets2'].get.responses['200'].content['application/json'].schema",
		canonical,
		"$.paths['/pets'].get.responses['200'].content['application/json'].schema",
		"$.paths['/pets2'].get.responses['200'].content['application/json'].schema",
	})

	assert.Equal(t, canonical, primaryPath)
	assert.Equal(t, []string{
		"$.components.schemas['Pet']",
		"$.paths['/pets'].get.responses['200'].content['application/json'].schema",
		"$.paths['/pets2'].get.responses['200'].content['application/json'].schema",
	}, allPaths)
}

func TestBuildStablePrimaryAndPaths_FallsBackToCanonical(t *testing.T) {
	canonical := "$.components.schemas['Pet']"
	primaryPath, allPaths := buildStablePrimaryAndPaths(canonical, nil)

	assert.Equal(t, canonical, primaryPath)
	assert.Equal(t, []string{canonical}, allPaths)
}

func TestBuildStablePrimaryAndPaths_NonComponentCanonicalIsSortedCandidate(t *testing.T) {
	canonical := "$.paths['/v1/resource'].get.responses['500'].content['*/*'].schema.properties['error-code']"
	primaryPath, allPaths := buildStablePrimaryAndPaths(canonical, []string{
		"$.paths['/v1/resource'].get.responses['404'].content['*/*'].schema.properties['error-code']",
		"$.paths['/v1/resource'].get.responses['400'].content['*/*'].schema.properties['error-code']",
	})

	assert.Equal(t, "$.paths['/v1/resource'].get.responses['400'].content['*/*'].schema.properties['error-code']", primaryPath)
	assert.Equal(t, []string{
		"$.paths['/v1/resource'].get.responses['400'].content['*/*'].schema.properties['error-code']",
		"$.paths['/v1/resource'].get.responses['404'].content['*/*'].schema.properties['error-code']",
		"$.paths['/v1/resource'].get.responses['500'].content['*/*'].schema.properties['error-code']",
	}, allPaths)
}
