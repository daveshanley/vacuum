// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley.
// SPDX-License-Identifier: MIT

function runRule(input) {

    // create an array to hold the results
    let results = vacuum_truthy(input, context);

    results.push({
        message: "this is a message, added after truthy was called",
    });

    // return results.
    return results;
}

