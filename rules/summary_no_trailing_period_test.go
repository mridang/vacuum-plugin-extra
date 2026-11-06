package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSummaryNoTrailingPeriod(t *testing.T) {
	t.Parallel()

	rule := rules.SummaryNoTrailingPeriod()

	t.Run("flags a summary ending with a period", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List pets.
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.summary", results[0].Path)
	})

	t.Run("accepts a summary without a trailing period", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores operations without a summary", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("tolerates trailing whitespace after the period", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      summary: "Create a pet. "
      responses:
        "201":
          description: created
`), nil, "$.paths['/pets'].post.summary")
	})
}
