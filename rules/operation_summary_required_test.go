package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationSummaryRequired(t *testing.T) {
	t.Parallel()

	rule := rules.OperationSummaryRequired()

	t.Run("flags missing and empty summaries", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
  /b:
    get:
      summary: "  "
      responses:
        "200":
          description: ok
`), nil, "$.paths['/a'].get", "$.paths['/b'].get.summary")
	})

	t.Run("accepts operations with a summary", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /a:
    get:
      summary: List a
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags only the operation lacking a summary", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /a:
    get:
      summary: ok
      responses:
        "200":
          description: ok
    post:
      responses:
        "200":
          description: ok
`), nil, "$.paths['/a'].post")
	})
}
