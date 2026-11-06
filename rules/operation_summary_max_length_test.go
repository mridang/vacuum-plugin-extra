package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationSummaryMaxLength(t *testing.T) {
	t.Parallel()

	rule := rules.OperationSummaryMaxLength()

	t.Run("flags a summary above the default limit", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: `+strings.Repeat("a", 81)+`
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/pets'].get.summary"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "81")
	})

	t.Run("accepts a summary at the limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: `+strings.Repeat("a", 80)+`
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours a lower limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxLength": 5}, "$.paths['/pets'].get.summary")
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
`), ruletest.Options{"maxLength": 1})
	})
}
