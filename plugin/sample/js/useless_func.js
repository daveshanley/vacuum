// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley.
// SPDX-License-Identifier: MIT

function getSchema() {
    return {
        "name": "useless_func_modified_name",
        "properties": [
            {
                "name": "mickey",
                "description": "a mouse"
            }
        ],
    };
}

function runRule() {
    return [
        {
            message: "this is a message",
        },
        {
            message: "this is another message",
        }
    ]
}

