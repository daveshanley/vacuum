package rulesets

import (
	"context"
	"fmt"
	"github.com/pb33f/testify/require"
	"path/filepath"
	"sync"
	"testing"
)

func TestCapturedReaderPreservesNestedLocalExtends(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		filepath.Join(dir, "root.yaml"):              "extends: [./nested/child.yaml]\nrules: {}\n",
		filepath.Join(dir, "nested/child.yaml"):      "extends: [./grandchild.yaml]\nrules: {}\n",
		filepath.Join(dir, "nested/grandchild.yaml"): "rules:\n  captured:\n    given: $.info\n    severity: error\n    then: {function: defined}\n",
	}
	var mu sync.Mutex
	reads := map[string]int{}
	reader := func(path string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		reads[path]++
		if data, ok := files[path]; ok {
			return []byte(data), nil
		}
		return nil, fmt.Errorf("uncaptured: %s", path)
	}
	loaded, err := LoadLocalRuleSetWithReader(context.Background(), filepath.Join(dir, "root.yaml"), reader)
	require.NoError(t, err)
	expanded := BuildDefaultRuleSets().GenerateRuleSetFromSuppliedRuleSet(loaded)
	require.NoError(t, expanded.LoadError())
	require.NotNil(t, expanded.Rules["captured"])
	mu.Lock()
	defer mu.Unlock()
	for path := range files {
		require.Equal(t, 1, reads[path], path)
	}
}
