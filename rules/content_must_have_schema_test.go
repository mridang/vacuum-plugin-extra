package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestContentMustHaveSchema(t *testing.T) {
	t.Parallel()

	rule := rules.ContentMustHaveSchema()

	t.Run("flags content entries without a schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json: {}
      parameters:
        - name: filter
          in: query
          content:
            application/json:
              example: x
      responses:
        "200":
          description: ok
          content:
            application/json:
              example: y
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].post.requestBody.content['application/json']",
			"$.paths['/pets'].post.parameters[0].content['application/json']",
			"$.paths['/pets'].post.responses['200'].content['application/json']",
		}, ruletest.Paths(results))
	})

	t.Run("flags components", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  requestBodies:
    Body:
      content:
        text/plain: {}
  responses:
    Ok:
      description: ok
      content:
        text/plain: {}
      headers:
        X-Data:
          content:
            text/plain: {}
`), nil,
			"$.components.requestBodies.Body.content['text/plain']",
			"$.components.responses.Ok.content['text/plain']",
			"$.components.responses.Ok.headers['X-Data'].content['text/plain']",
		)
	})

	t.Run("accepts entries with a schema", func(t *testing.T) {
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
        "404":
          $ref: "#/components/responses/NotFound"
components:
  responses:
    NotFound:
      description: missing
`), nil)
	})
}
