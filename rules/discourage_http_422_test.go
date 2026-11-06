package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDiscourageHTTP422(t *testing.T) {
	t.Parallel()

	rule := rules.DiscourageHTTP422()

	t.Run("flags a 422 response", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "201":
          description: created
        "422":
          description: invalid
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].post.responses['422']", results[0].Path)
		require.Contains(t, results[0].Message, "400")
	})

	t.Run("accepts a 400 response", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "201":
          description: created
        "400":
          description: invalid
`), nil)
	})

	t.Run("reports each operation separately", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      responses:
        "422":
          description: invalid
    put:
      responses:
        "422":
          description: invalid
`), nil, "$.paths['/pets'].post.responses['422']", "$.paths['/pets'].put.responses['422']")
	})

	t.Run("ignores operations without responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get: {}
`), nil)
	})
}
