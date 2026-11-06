package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoWildcardMediaType(t *testing.T) {
	t.Parallel()

	rule := rules.NoWildcardMediaType()

	t.Run("flags wildcard keys in requests, responses and components", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          "*/*":
            schema: {type: object}
      responses:
        "200":
          description: ok
          content:
            "*/*":
              schema: {type: object}
            image/*:
              schema: {type: string}
components:
  requestBodies:
    Body:
      content:
        "*/*": {}
`), nil,
			"$.paths['/pets'].post.requestBody.content",
			"$.paths['/pets'].post.responses['200'].content",
			"$.components.requestBodies.Body.content")
	})

	t.Run("accepts concrete and partial wildcard types", func(t *testing.T) {
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
              schema: {type: object}
            image/*:
              schema: {type: string}
`), nil)
	})

	t.Run("flags wildcard parameter content", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          content:
            "*/*":
              schema: {type: string}
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].get.parameters[0].content")
	})
}
