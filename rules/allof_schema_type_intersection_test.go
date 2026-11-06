package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAllofSchemaTypeIntersection(t *testing.T) {
	t.Parallel()

	rule := rules.AllofSchemaTypeIntersection()

	t.Run("flags a schema whose allOf branch has a conflicting type", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      allOf:
        - type: string
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Pet.allOf", results[0].Path)
		require.Contains(t, results[0].Message, "no type in common")
	})

	t.Run("flags two conflicting branches of an untyped schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      allOf:
        - type: string
        - type: integer
`), nil, "$.components.schemas.Pet.allOf")
	})

	t.Run("resolves a referenced branch", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Name:
      type: string
    Pet:
      type: integer
      allOf:
        - $ref: "#/components/schemas/Name"
`), nil, "$.components.schemas.Pet.allOf")
	})

	t.Run("accepts matching types and untyped branches", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      allOf:
        - type: object
        - properties:
            name:
              type: string
`), nil)
	})

	t.Run("treats integer as a subset of number", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Count:
      type: number
      allOf:
        - type: integer
`), nil)
	})

	t.Run("supports OpenAPI 3.1 type lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Value:
      type: [string, "null"]
      allOf:
        - type: string
`), nil)

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Value:
      type: [string, "null"]
      allOf:
        - type: integer
`), nil, "$.components.schemas.Value.allOf")
	})
}
