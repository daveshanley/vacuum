function runRule(input) {
    const numPaths = Object.keys(input).length;
    if (numPaths > 1) {
        return [
            {
                message: 'more than a single path exists, there are ' + numPaths
            }
        ];
    }
}