package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestExplodedObjectQueryRequiredProperty(t *testing.T) {
	t.Parallel()

	rule := rules.ExplodedObjectQueryRequiredProperty()

	t.Run("flags a required exploded object without required properties", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: filter
          in: query
          required: true
          schema:
            type: object
            properties:
              color:
                type: string
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/search'].get.parameters[0]", results[0].Path)
		require.Contains(t, results[0].Message, "filter")
	})

	t.Run("accepts an object with a required property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: filter
          in: query
          required: true
          schema:
            type: object
            required: [color]
            properties:
              color:
                type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores optional, non-exploded and deepObject parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: a
          in: query
          schema:
            type: object
        - name: b
          in: query
          required: true
          explode: false
          schema:
            type: object
        - name: c
          in: query
          required: true
          style: deepObject
          schema:
            type: object
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("follows $ref schemas and allOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /search:
    get:
      parameters:
        - name: bad
          in: query
          required: true
          schema:
            $ref: "#/components/schemas/Bad"
        - name: good
          in: query
          required: true
          schema:
            allOf:
              - $ref: "#/components/schemas/Good"
            type: object
      responses:
        "200":
          description: ok
components:
  schemas:
    Bad:
      type: object
      properties:
        x:
          type: string
    Good:
      required: [x]
`), nil, "$.paths['/search'].get.parameters[0]")
	})
}
