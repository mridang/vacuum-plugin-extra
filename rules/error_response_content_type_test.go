package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestErrorResponseContentType(t *testing.T) {
	t.Parallel()

	rule := rules.ErrorResponseContentType()

	t.Run("flags error responses without a json entry", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            text/plain:
              schema:
                type: string
        "404":
          description: not found
          content:
            text/plain:
              schema:
                type: string
        4XX:
          description: client error
          content:
            application/xml:
              schema:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['404'].content",
			"$.paths['/pets'].get.responses['4XX'].content",
		}, ruletest.Paths(results))
	})

	t.Run("accepts json entries with parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "400":
          description: bad
          content:
            application/json; charset=utf-8:
              schema:
                type: object
        "500":
          description: boom
          content:
            application/json:
              schema:
                type: object
`), nil)
	})

	t.Run("ignores bodiless error responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "404":
          description: not found
`), nil)
	})

	t.Run("resolves referenced responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "500":
          $ref: "#/components/responses/Boom"
components:
  responses:
    Boom:
      description: boom
      content:
        text/html:
          schema:
            type: string
`), nil, "$.paths['/pets'].get.responses['500'].content")
	})
}
