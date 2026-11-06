package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestEmptyRequiredOrEnum(t *testing.T) {
	t.Parallel()

	rule := rules.EmptyRequiredOrEnum()

	t.Run("flags empty and duplicated lists", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      required: []
      properties:
        kind:
          type: string
          enum: [cat, cat]
        size:
          type: string
          enum: []
    Dup:
      type: object
      required: [a, b, a]
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet.required",
			"$.components.schemas.Pet.properties.kind.enum[1]",
			"$.components.schemas.Pet.properties.size.enum",
			"$.components.schemas.Dup.required[2]",
		}, ruletest.Paths(results))
	})

	t.Run("accepts valid lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      properties:
        kind:
          type: string
          enum: [cat, dog]
        code:
          type: integer
          enum: [1, "1"]
`), nil)
	})

	t.Run("ignores the boolean required flag of parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})
}
