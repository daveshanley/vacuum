// Copyright 2023 Princess Beef Heavy Industries / Dave Shanley.
// SPDX-License-Identifier: MIT

function getSchema() {
    return {
        "name": "checkForNameAndId",
        "description": "Checks that input has the correct name and id values"
    };
}

function runRule(input) {

    // create an array to hold the results
    let results = [];

    // check if the input.name and input.id are the correct values
    if (input.name != "some_name" || input.id != "some_id") {

        // add a new failure result to the results array
        results.push({
            message: "name '" + input.name + "' and id '" + input.id + "' are not 'some_name' or 'some_id'",
        });
    }

    // return results.
    return results;
}

