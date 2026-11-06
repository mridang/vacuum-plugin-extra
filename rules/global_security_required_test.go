package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestGlobalSecurityRequired(t *testing.T) {
	t.Parallel()

	rule := rules.GlobalSecurityRequired()

	t.Run("flags a missing security property", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`paths: {}`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$", results[0].Path)
	})

	t.Run("flags an empty list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
security: []
paths: {}
`), nil, "$.security")
	})

	t.Run("flags entries that are not objects", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
security:
  - apiKey: []
  - plain
paths: {}
`), nil, "$.security[1]")
	})

	t.Run("accepts a non-empty list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
security:
  - apiKey: []
paths: {}
`), nil)
	})
}
