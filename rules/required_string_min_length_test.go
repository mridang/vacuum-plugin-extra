package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequiredStringMinLength(t *testing.T) {
	t.Parallel()

	rule := rules.RequiredStringMinLength()

	t.Run("flags required strings without minLength", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      required: [name, age, nick, kind]
      properties:
        name:
          type: string
        age:
          type: integer
        nick:
          $ref: "#/components/schemas/Nick"
        kind:
          type: string
          enum: [cat, dog]
        optional:
          type: string
    Nick:
      type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet.properties.name",
			"$.components.schemas.Pet.properties.nick",
		}, ruletest.Paths(results))
	})

	t.Run("accepts minLength including zero", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      required: [name, note]
      properties:
        name:
          type: string
          minLength: 1
        note:
          type: string
          minLength: 0
`), nil)
	})

	t.Run("handles 3.1 type lists and missing properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      required: [name, ghost]
      properties:
        name:
          type: [string, "null"]
`), nil, "$.components.schemas.Pet.properties.name")
	})
}
