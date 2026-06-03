// Copyright 2020-2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// https://quobix.com/vacuum/ | https://pb33f.io
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	asyncapi_context "github.com/daveshanley/vacuum/asyncapi"
)

func rejectAsyncAPIForOpenAPICommand(command string, specBytes []byte) error {
	isAsyncAPI, err := asyncapi_context.IsDocument(specBytes)
	if err != nil {
		if !asyncapi_context.HasMarker(specBytes) {
			return nil
		}
		return asyncAPIUnsupportedCommandError(command)
	}
	if !isAsyncAPI {
		return nil
	}
	return asyncAPIUnsupportedCommandError(command)
}

func asyncAPIUnsupportedCommandError(command string) error {
	return fmt.Errorf("`vacuum %s` only supports OpenAPI documents; AsyncAPI support is currently limited to linting", command)
}
