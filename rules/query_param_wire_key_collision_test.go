package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestQueryParamWireKeyCollision(t *testing.T) {
	t.Parallel()

	rule := rules.QueryParamWireKeyCollision()

	t.Run("flags an exploded property that collides with a plain parameter", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: color
          in: query
          schema:
            type: string
        - name: filter
          in: query
          schema:
            type: object
            properties:
              color:
                type: integer
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/search'].get.parameters[1]"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "color")
	})

	t.Run("accepts compatible collisions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: color
          in: query
          schema:
            type: string
        - name: filter
          in: query
          schema:
            type: object
            properties:
              color:
                type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("accepts non-exploded and deepObject objects", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: color
          in: query
          schema:
            type: string
        - name: a
          in: query
          explode: false
          schema:
            type: object
            properties:
              color:
                type: integer
        - name: b
          in: query
          style: deepObject
          schema:
            type: object
            properties:
              color:
                type: integer
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags two exploded objects with the same property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - $ref: "#/components/parameters/A"
        - $ref: "#/components/parameters/B"
      responses:
        "200":
          description: ok
components:
  parameters:
    A:
      name: a
      in: query
      schema:
        type: object
        properties:
          id:
            type: string
    B:
      name: b
      in: query
      schema:
        type: object
        properties:
          id:
            type: string
            format: uuid
`), nil, "$.paths['/search'].get.parameters[1]")
	})
}
