/**
 * Example: Sentiment Analysis Validation (Batch Mode)
 *
 * This example demonstrates how to call an external API from a vacuum
 * JavaScript custom function using the built-in fetch() API.
 *
 * In batch mode (functionOptions.batch: true), this function receives ALL
 * matched descriptions at once, allowing a single API call instead of one
 * call per description. For a spec with 29 descriptions, this means 1 API
 * call instead of 29.
 *
 * Compare with sentiment_check.js which makes one API call per description.
 *
 * Input format (batch mode):
 *   [{value: "description1", index: 0}, {value: "description2", index: 1}, ...]
 *
 * Output format (batch mode REQUIRES including the input object):
 *   [{message: "error", input: inputs[i]}]
 *
 * The input object contains the index field we use to map back to the
 * correct node. This is deterministic and required for batch mode.
 *
 * Usage:
 *   vacuum lint spec.yaml -r rulesets/examples/sentiment-check-batch.yaml -f plugin/sample/js
 *
 * (see model/test_files/sad-spec.yaml for a test spec)
 */

function getSchema() {
    return {
        name: "sentimentCheckBatch",
        description: "Checks if API descriptions are too negative using sentiment analysis (batch mode)"
    };
}

/**
 * Main validation function (batch mode).
 * Receives all descriptions at once from the $..description JSONPath.
 */
async function runRule(inputs) {
    // build request body and a lookup map: id -> original input object
    var requestBody = [];
    var idToInput = {};

    for (var i = 0; i < inputs.length; i++) {
        var input = inputs[i];
        if (input.value && typeof input.value === "string") {
            requestBody.push({
                id: input.index,
                description: input.value
            });
            // Store original input for result mapping (required for batch mode)
            idToInput[input.index] = input;
        }
    }

    // nothing to check
    if (requestBody.length === 0) {
        return [];
    }

    // single API call with all descriptions
    var response = await fetch("https://api.pb33f.io/sentiment-check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(requestBody)
    });

    if (!response.ok) {
        console.log("Sentiment API error: " + response.status);
        return [];
    }

    var data = await response.json();

    if (!data.results || data.results.length === 0) {
        return [];
    }

    // map API results back to vacuum results
    var results = [];
    for (var j = 0; j < data.results.length; j++) {
        var result = data.results[j];

        // only report violations (tooSad = true)
        if (!result.tooSad) {
            continue;
        }

        // format negative words as a quoted list
        var words = result.negativeWords || [];
        var wordList = words.map(function(w) { return "'" + w + "'"; }).join(", ");

        var message = "description is too negative";
        if (wordList) {
            message += ", words like " + wordList + " are lowering your overall tone";
        } else {
            message += ", the combination of words used are too dreary";
        }

        // return result with input for deterministic node mapping (required in batch mode)
        // vacuum uses input.index to map back to the correct node
        results.push({
            message: message,
            input: idToInput[result.id]
        });
    }

    return results;
}
