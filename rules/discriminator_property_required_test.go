package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDiscriminatorPropertyRequired(t *testing.T) {
	t.Parallel()

	rule := rules.DiscriminatorPropertyRequired()

	t.Run("flags a discriminator property that is not required", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
      required: [name]
      properties:
        kind:
          type: string
        name:
          type: string
`), nil)

		require.Equal(t, []string{"$.components.schemas.Pet.discriminator.propertyName"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "kind")
	})

	t.Run("flags a schema without any required list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
`), nil, "$.components.schemas.Pet.discriminator.propertyName")
	})

	t.Run("accepts a required discriminator property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      discriminator:
        propertyName: kind
      properties:
        kind:
          type: string
`), nil)
	})

	t.Run("accepts a required property in an allOf member", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Base:
      type: object
      required: [kind]
      properties:
        kind:
          type: string
    Pet:
      discriminator:
        propertyName: kind
      allOf:
        - $ref: "#/components/schemas/Base"
`), nil)
	})

	t.Run("ignores schemas without a discriminator", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        kind:
          type: string
`), nil)
	})
}
