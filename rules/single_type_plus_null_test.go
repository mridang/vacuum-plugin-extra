package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSingleTypePlusNull(t *testing.T) {
	t.Parallel()

	rule := rules.SingleTypePlusNull()

	t.Run("flags several types besides null", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: [string, integer]
    B:
      type: [string, integer, "null"]
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.A.type",
			"$.components.schemas.B.type",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "single type")
	})

	t.Run("accepts one type with or without null", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: [string]
    B:
      type: [string, "null"]
    C:
      type: integer
`), nil)
	})

	t.Run("checks nested schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age:
          type: [integer, number]
`), nil, "$.components.schemas.Pet.properties.age.type")
	})

	t.Run("is not applied to OpenAPI 3.0 documents", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: [string, integer]
`), nil)
	})
}
