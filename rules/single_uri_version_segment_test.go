package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSingleURIVersionSegment(t *testing.T) {
	t.Parallel()

	rule := rules.SingleURIVersionSegment()

	t.Run("flags a path with two version segments", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /v1/orders/v2/items:
    get:
      responses:
        "200":
          description: ok
  /v1.2/orders/V3:
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/v1/orders/v2/items']",
			"$.paths['/v1.2/orders/V3']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "version segment")
	})

	t.Run("accepts one version segment and lookalike names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /v1/orders:
    get:
      responses:
        "200":
          description: ok
  /orders/vat/v2:
    get:
      responses:
        "200":
          description: ok
  /v1/videos/v2k:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}
