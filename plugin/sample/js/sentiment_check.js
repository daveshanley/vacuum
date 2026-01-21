/**
 * Example: Sentiment Analysis Validation (Per-Node Mode)
 *
 * This example demonstrates how to call an external API from a vacuum
 * JavaScript custom function using the built-in fetch() API.
 *
 * In per-node mode (the default), this function is called ONCE for each
 * matched description. For a spec with 29 descriptions, this means 29
 * separate API calls.
 *
 * Compare with sentiment_check_batch.js which batches all descriptions
 * into a single API call for better performance.
 *
 * Usage:
 *   vacuum lint spec.yaml -r rulesets/examples/sentiment-check.yaml -f plugin/sample/js
 *
 * (see model/test_files/sad-spec.yaml for a test spec)
 */

function getSchema() {
    return {
        name: "sentimentCheck",
        description: "Checks if API descriptions are too negative using sentiment analysis"
    };
}

/**
 * Main validation function (per-node mode).
 * Called once for each description matched by $..description JSONPath.
 */
async function runRule(input) {
    // Skip non-string values
    if (!input || typeof input !== "string") {
        return [];
    }

    // API call for this single description
    var response = await fetch("https://api.pb33f.io/sentiment-check", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify([{
            id: 0,
            description: input
        }])
    });

    if (!response.ok) {
        console.log("Sentiment API error: " + response.status);
        return [];
    }

    var data = await response.json();

    if (!data.results || data.results.length === 0) {
        return [];
    }

    var result = data.results[0];

    // Only report violations (tooSad = true)
    if (!result.tooSad) {
        return [];
    }

    // Format negative words as a quoted list
    var words = result.negativeWords || [];
    var wordList = words.map(function(w) { return "'" + w + "'"; }).join(", ");

    var message = "description is too negative";
    if (wordList) {
        message += ", words like " + wordList + " are lowering your overall tone";
    } else {
        message += ", the combination of words used are too dreary";
    }

    return [{ message: message }];
}
