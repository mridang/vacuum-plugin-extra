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

	t.Run("ignores null branches by default", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      oneOf:
        - type: string
        - type: "null"
`), nil)
	})

	t.Run("drops null from a type list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      oneOf:
        - type: [string, "null"]
        - type: string
`), nil)
	})

	t.Run("still flags a real mix next to a null branch", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Id:
      oneOf:
        - type: string
        - type: integer
        - type: "null"
`), nil)

		require.Equal(t, []string{"all branches of the composition must have the same type, found: integer, string"},
			ruletest.Messages(results))
	})

	t.Run("compares null as a type when ignoreNull is false", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      oneOf:
        - type: string
        - type: "null"
`), ruletest.Options{"ignoreNull": false})

		require.Equal(t, []string{"all branches of the composition must have the same type, found: null, string"},
			ruletest.Messages(results))
	})

	t.Run("does not compare not branches with the composition", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      type: string
      oneOf:
        - type: string
          maxLength: 3
        - type: string
          minLength: 1
      not:
        type: integer
`), nil)
	})
}
