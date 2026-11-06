package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestCompositionBranchesSameType(t *testing.T) {
	t.Parallel()

	rule := rules.CompositionBranchesSameType()

	t.Run("flags branches with different types", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Id:
      oneOf:
        - type: string
        - type: integer
`), nil)

		require.Equal(t, []string{"$.components.schemas.Id"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "integer, string")
	})

	t.Run("follows references to the leaves", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Wrapper:
      anyOf:
        - $ref: "#/components/schemas/Name"
        - $ref: "#/components/schemas/Count"
    Name:
      type: string
    Count:
      type: integer
`), nil, "$.components.schemas.Wrapper")
	})

	t.Run("flattens nested compositions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Mixed:
      allOf:
        - type: object
        - oneOf:
            - type: object
            - type: array
              items:
                type: string
`), nil, "$.components.schemas.Mixed", "$.components.schemas.Mixed.allOf[1]")
	})

	t.Run("accepts branches of the same type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      oneOf:
        - $ref: "#/components/schemas/Cat"
        - $ref: "#/components/schemas/Dog"
    Cat:
      type: object
    Dog:
      type: object
`), nil)
	})

	t.Run("ignores untyped branches", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Node:
      oneOf:
        - description: no type
        - type: object
`), nil)
	})
}
