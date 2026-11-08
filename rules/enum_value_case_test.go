package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestEnumValueCase(t *testing.T) {
	t.Parallel()

	rule := rules.EnumValueCase()

	t.Run("flags values that are neither snake nor screaming snake", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Status:
      type: string
      enum:
        - in-progress
        - Done
        - open
        - IN_REVIEW
        - bad__double
        - _lead
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Status.enum[0]",
			"$.components.schemas.Status.enum[1]",
			"$.components.schemas.Status.enum[4]",
			"$.components.schemas.Status.enum[5]",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "snake_case")
	})

	t.Run("accepts snake case and screaming snake case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Status:
      type: string
      enum: [in_progress, done, IN_REVIEW, V2_BETA, ONE]
`), nil)
	})

	t.Run("ignores null and non string values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Code:
      type: [integer, "null"]
      enum: [1, 2, null]
`), nil)
	})

	t.Run("checks inline schemas in parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: sort
          in: query
          schema:
            type: string
            enum: [createdAt]
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].get.parameters[0].schema.enum[0]")
	})

	t.Run("honours the casing option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
components:
  schemas:
    Status:
      type: string
      enum: [done, DONE]
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"casing": "snake_case"},
			"$.components.schemas.Status.enum[1]")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"casing": []string{"SCREAMING_SNAKE_CASE"}},
			"$.components.schemas.Status.enum[0]")
	})
}
