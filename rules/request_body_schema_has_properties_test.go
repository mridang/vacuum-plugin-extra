package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequestBodySchemaHasProperties(t *testing.T) {
	t.Parallel()

	rule := rules.RequestBodySchemaHasProperties()

	t.Run("flags a bare object schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
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
          description: created
`), nil)

		require.Equal(
			t,
			[]string{"$.paths['/pets'].post.requestBody.content['application/json'].schema"},
			ruletest.Paths(results),
		)
		require.Contains(t, results[0].Message, "properties")
	})

	t.Run("accepts a schema with properties", func(t *testing.T) {
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
              properties:
                name:
                  type: string
      responses:
        "201":
          description: created
`), nil)
	})

	t.Run("resolves references and checks component bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/vnd.api+json:
            schema:
              $ref: "#/components/schemas/Empty"
      responses:
        "201":
          description: created
components:
  schemas:
    Empty:
      type: object
  requestBodies:
    Owner:
      content:
        application/json:
          schema:
            description: nothing
`), nil,
			"$.paths['/pets'].post.requestBody.content['application/vnd.api+json'].schema",
			"$.components.requestBodies.Owner.content['application/json'].schema",
		)
	})

	t.Run("ignores other media types, arrays and composed schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          text/plain:
            schema:
              type: object
          application/json:
            schema:
              type: array
              items:
                type: string
      responses:
        "201":
          description: created
    put:
      requestBody:
        content:
          application/json:
            schema:
              allOf:
                - type: object
      responses:
        "200":
          description: ok
`), nil)
	})
}
