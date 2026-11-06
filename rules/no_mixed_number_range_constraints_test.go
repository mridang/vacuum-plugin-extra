package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoMixedNumberRangeConstraints(t *testing.T) {
	t.Parallel()

	rule := rules.NoMixedNumberRangeConstraints()

	t.Run("flags numeric exclusive bounds next to inclusive ones", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Age:
      type: integer
      maximum: 10
      exclusiveMaximum: 10
      minimum: 1
      exclusiveMinimum: 0
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Age.exclusiveMaximum",
			"$.components.schemas.Age.exclusiveMinimum",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"Schema should not have both `maximum` and `exclusiveMaximum`. Use one or the other.",
		)
	})

	t.Run("accepts a single bound per side", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Age:
      type: integer
      exclusiveMaximum: 10
      minimum: 1
`), nil)
	})

	t.Run("accepts the OpenAPI 3.0 boolean form", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Age:
      type: integer
      maximum: 10
      exclusiveMaximum: true
      minimum: 1
      exclusiveMinimum: false
`), nil)
	})

	t.Run("checks nested property schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        age:
          type: number
          minimum: 0
          exclusiveMinimum: 0
`), nil, "$.components.schemas.Pet.properties.age.exclusiveMinimum")
	})

	t.Run("reports the numeric form on 3.0 documents and ignores strings", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Numeric:
      type: number
      minimum: 1.5
      exclusiveMinimum: 1.5
    Quoted:
      type: number
      minimum: 1
      exclusiveMinimum: "1"
`), nil, "$.components.schemas.Numeric.exclusiveMinimum")
	})
}
