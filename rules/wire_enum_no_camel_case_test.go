package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWireEnumNoCamelCase(t *testing.T) {
	t.Parallel()

	rule := rules.WireEnumNoCamelCase()

	t.Run("flags camelCase enum values", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Status:
      type: string
      enum: [inProgress, done, created_atUTC, DONE_NOW, in-progress]
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Status.enum[0]",
			"$.components.schemas.Status.enum[2]",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "camelCase")
	})

	t.Run("accepts snake, kebab and screaming case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Status:
      type: string
      enum: [in_progress, DONE, in-progress, Done]
`), nil)
	})

	t.Run("checks nested schemas and ignores numbers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        kind:
          type: string
          enum: [bigDog]
        level:
          type: integer
          enum: [1, 2]
`), nil, "$.components.schemas.Pet.properties.kind.enum[0]")
	})
}
