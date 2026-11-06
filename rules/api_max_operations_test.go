package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAPIMaxOperations(t *testing.T) {
	t.Parallel()

	rule := rules.APIMaxOperations()

	const twoOperations = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
    post:
      responses:
        "201":
          description: created
`

	t.Run("flags a document above the limit", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(twoOperations), ruletest.Options{"maxOperations": 1})

		require.Len(t, results, 1)
		require.Equal(t, "$.paths", results[0].Path)
		require.Contains(t, results[0].Message, "2 operations")
	})

	t.Run("accepts a document at the limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(twoOperations), ruletest.Options{"maxOperations": 2})
	})

	t.Run("accepts a small document with the default limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(twoOperations), nil)
	})

	t.Run("does not count non method keys", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    summary: pets
    parameters: []
    get:
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxOperations": 1})
	})

	t.Run("a non positive limit disables the rule", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(twoOperations), ruletest.Options{"maxOperations": 0})
	})
}
