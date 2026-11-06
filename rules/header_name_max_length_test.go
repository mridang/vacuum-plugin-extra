package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestHeaderNameMaxLength(t *testing.T) {
	t.Parallel()

	rule := rules.HeaderNameMaxLength()

	t.Run("flags a header name over the default limit", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Some-Extremely-Long-Header-Name-That-Keeps-Going-And-Going:
              schema:
                type: string
            X-Request-Id:
              schema:
                type: string
`), nil)

		require.Len(t, results, 1)
		require.Equal(
			t,
			"$.paths['/pets'].get.responses['200'].headers['X-Some-Extremely-Long-Header-Name-That-Keeps-Going-And-Going']",
			results[0].Path,
		)
	})

	t.Run("accepts short header names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Request-Id:
              schema:
                type: string
`), nil)
	})

	t.Run("honours the maxLength option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-Request-Id:
              schema:
                type: string
`), ruletest.Options{"maxLength": 5}, "$.paths['/pets'].get.responses['200'].headers['X-Request-Id']")
	})

	t.Run("checks component responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  responses:
    Ok:
      description: ok
      headers:
        X-Rate-Limit-Remaining-Requests-In-The-Current-Window-Period:
          schema:
            type: integer
`), nil, "$.components.responses.Ok.headers['X-Rate-Limit-Remaining-Requests-In-The-Current-Window-Period']")
	})
}
