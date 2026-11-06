package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestResponseRequestContentRequired(t *testing.T) {
	t.Parallel()

	rule := rules.ResponseRequestContentRequired()

	t.Run("flags bodies and responses without content", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        description: a pet
      responses:
        "201":
          description: created
        "204":
          description: none
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.requestBody",
			"$.paths['/pets'].post.responses['201']",
		}, ruletest.Paths(results))
	})

	t.Run("accepts operations with content everywhere", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          $ref: "#/components/responses/Created"
components:
  responses:
    Created:
      description: created
      content:
        application/json:
          schema:
            type: object
`), nil)
	})

	t.Run("flags a referenced response at the operation", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "404":
          $ref: "#/components/responses/Missing"
components:
  responses:
    Missing:
      description: missing
`), nil, "$.paths['/pets'].get.responses['404']")
	})

	t.Run("exempts head operations and informational codes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    head:
      responses:
        "200":
          description: ok
    get:
      responses:
        "100":
          description: continue
        "304":
          description: not modified
`), nil)
	})

	t.Run("honours ignoreCodes", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    delete:
      responses:
        "202":
          description: accepted
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].delete.responses['202']")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"ignoreCodes": []any{"202"}})
	})
}
