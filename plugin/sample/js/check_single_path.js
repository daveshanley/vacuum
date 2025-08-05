function getSchema() {
    return {
        "name": "checkSinglePath",
        "description": "Checks that there is only a single path defined in the OpenAPI spec"
    };
}

function runRule(input) {
    // input should be the paths object from the OpenAPI spec
    if (!input || typeof input !== 'object') {
        return [];
    }
    
    const numPaths = Object.keys(input).length;
    if (numPaths > 1) {
        return [
            {
                message: 'More than a single path exists, found ' + numPaths + ' paths'
            }
        ];
    }
    
    // Return empty array when rule passes (no violations)
    return [];
}