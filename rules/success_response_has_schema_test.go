package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSuccessResponseHasSchema(t *testing.T) {
	t.Parallel()

	rule := rules.SuccessResponseHasSchema()

	t.Run("flags a success response without content or schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
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
          content:
            application/json:
              example: {}
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200']",
			"$.paths['/pets'].post.responses['201']",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "schema")
	})

	t.Run("accepts a response with a schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
`), nil)
	})

	t.Run("ignores 204 responses, head operations and error responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    delete:
      responses:
        "204":
          description: gone
        "404":
          description: missing
    head:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours ignoredCodes and only examines the first success response", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
        "206":
          description: partial
`)

		ruletest.RequirePaths(t, rule, document, nil, "$.paths['/pets'].get.responses['200']")
		ruletest.RequireClean(t, rule, document, ruletest.Options{"ignoredCodes": []any{"200"}})
	})
}
