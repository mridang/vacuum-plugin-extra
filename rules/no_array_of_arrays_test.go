package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoArrayOfArrays(t *testing.T) {
	t.Parallel()

	rule := rules.NoArrayOfArrays()

	t.Run("flags inline nested arrays", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Matrix:
      type: array
      items:
        type: array
        items:
          type: number
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Matrix.items", results[0].Path)
	})

	t.Run("flags nested arrays reached through a ref", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Row:
      type: array
      items:
        type: number
    Matrix:
      type: array
      items:
        $ref: "#/components/schemas/Row"
`), nil, "$.components.schemas.Matrix.items")
	})

	t.Run("flags 3.1 type lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Matrix:
      type: [array, "null"]
      items:
        type: array
        items:
          type: number
`), nil, "$.components.schemas.Matrix.items")
	})

	t.Run("accepts arrays of objects", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    List:
      type: array
      items:
        type: object
        properties:
          values:
            type: array
            items:
              type: number
`), nil)
	})
}
