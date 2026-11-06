package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestArrayBoundsAndItemsConsistency(t *testing.T) {
	t.Parallel()

	rule := rules.ArrayBoundsAndItemsConsistency()

	t.Run("ignores constraint fragments below not", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Step:
      type: object
      not:
        properties:
          tags: {type: array, minItems: 1}
`), nil)
	})

	t.Run("flags a bare array", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tags:
      type: array
`), nil)

		require.Len(t, results, 3)

		for _, result := range results {
			require.Equal(t, "$.components.schemas.Tags", result.Path)
		}

		require.Contains(t, ruletest.Messages(results), "array schemas must specify an 'items' schema")
	})

	t.Run("accepts a bounded array with items", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tags:
      type: array
      minItems: 0
      maxItems: 10
      items:
        type: string
`), nil)
	})

	t.Run("flags minItems above maxItems", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tags:
      type: array
      minItems: 5
      maxItems: 2
      items:
        type: string
`), nil)

		require.Equal(t, []string{"'minItems' cannot be greater than 'maxItems'"}, ruletest.Messages(results))
	})

	t.Run("bounds are optional when requireBounds is false", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tags:
      type: array
      items:
        type: string
`), ruletest.Options{"requireBounds": false})
	})

	t.Run("still requires items when requireBounds is false", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
`), ruletest.Options{"requireBounds": false}, "$.paths['/pets'].get.responses['200'].content['application/json'].schema")
	})

	t.Run("finds attributes inside allOf members", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Tags:
      type: array
      allOf:
        - minItems: 1
          maxItems: 3
          items:
            type: string
`), nil)
	})
}
