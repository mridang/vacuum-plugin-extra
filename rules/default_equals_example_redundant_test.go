package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDefaultEqualsExampleRedundant(t *testing.T) {
	t.Parallel()

	rule := rules.DefaultEqualsExampleRedundant()

	t.Run("flags an example equal to the default", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Limit:
      type: integer
      default: 20
      example: 20
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Limit.example", results[0].Path)
	})

	t.Run("flags objects regardless of key order", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Point:
      type: object
      default: {x: 1, y: 2}
      example: {y: 2, x: 1}
`), nil, "$.components.schemas.Point.example")
	})

	t.Run("flags an entry of an examples list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      type: string
      default: pet
      examples: [dog, pet]
`), nil, "$.components.schemas.Name.examples[1]")
	})

	t.Run("accepts different values and missing defaults", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Limit:
      type: integer
      default: 20
      example: 50
    Other:
      type: string
      example: x
    Typed:
      type: string
      default: "20"
      example: 20
`), nil)
	})
}
