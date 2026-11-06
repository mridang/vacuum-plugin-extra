package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const errorResponseHasBodyDoc = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
        "404":
          description: missing
        4XX:
          $ref: "#/components/responses/Bare"
        "400":
          description: bad
          content:
            application/json:
              schema:
                type: object
components:
  responses:
    Bare:
      description: bare
      content:
        application/json: {}
`

func TestErrorResponseHasBody(t *testing.T) {
	t.Parallel()

	rule := rules.ErrorResponseHasBody()

	t.Run("flags 4xx responses without a schema body", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(errorResponseHasBodyDoc), nil,
			"$.paths['/pets'].get.responses['404']", "$.paths['/pets'].get.responses['4XX']")
	})

	t.Run("accepts bodies without a schema when not required", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(errorResponseHasBodyDoc),
			ruletest.Options{"requireSchema": false}, "$.paths['/pets'].get.responses['404']")
	})

	t.Run("accepts operations with only success responses", func(t *testing.T) {
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
}
