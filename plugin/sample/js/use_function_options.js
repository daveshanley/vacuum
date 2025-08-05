// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley.
// SPDX-License-Identifier: MIT

function getSchema() {
    return {
        "name": "useFunctionOptions",
        "description": "Demonstrates how to use function options passed from rulesets"
    };
}

function runRule(input) {
    // extract function options from context
    const functionOptions = context.ruleAction.functionOptions

    // check if the 'someOption' value is set in our options
    if (functionOptions.someOption) {
        return [
            {
                message: "someOption is set to " + functionOptions.someOption,
            }
        ];
    } else {
        return [
            {
                message: "someOption is not set",
            }
        ];
    }
}

