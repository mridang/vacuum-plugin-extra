package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDiscriminatorPropertyExists(t *testing.T) {
	t.Parallel()

	rule := rules.DiscriminatorPropertyExists()

	t.Run("flags a discriminator naming a missing property", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
      properties:
        name:
          type: string
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Pet.discriminator.propertyName", results[0].Path)
		require.Contains(t, results[0].Message, "kind")
	})

	t.Run("accepts a property defined directly", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
      properties:
        kind:
          type: string
`), nil)
	})

	t.Run("accepts a property defined through allOf and $ref", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Base:
      type: object
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

	t.Run("requires every oneOf member to define the property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Cat:
      type: object
      properties:
        kind:
          type: string
    Dog:
      type: object
      properties:
        bark:
          type: boolean
    Good:
      discriminator:
        propertyName: kind
      oneOf:
        - $ref: "#/components/schemas/Cat"
        - $ref: "#/components/schemas/Cat"
    Bad:
      discriminator:
        propertyName: kind
      oneOf:
        - $ref: "#/components/schemas/Cat"
        - $ref: "#/components/schemas/Dog"
`), nil, "$.components.schemas.Bad.discriminator.propertyName")
	})

	t.Run("flags a discriminator without propertyName", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        mapping: {}
`), nil, "$.components.schemas.Pet.discriminator.propertyName")
	})
}
