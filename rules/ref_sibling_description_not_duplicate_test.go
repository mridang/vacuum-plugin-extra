package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const refSiblingDescriptionNotDuplicateName = `
    Name:
      type: string
      description: The name   of the pet.
`

func TestRefSiblingDescriptionNotDuplicate(t *testing.T) {
	t.Parallel()

	rule := rules.RefSiblingDescriptionNotDuplicate()

	t.Run("flags a duplicated description in the allOf form", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:`+refSiblingDescriptionNotDuplicateName+`
    Pet:
      type: object
      properties:
        name:
          description: The name of the pet.
          allOf:
            - $ref: "#/components/schemas/Name"
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Pet.properties.name.description", results[0].Path)
	})

	t.Run("flags a duplicated description in the 3.1 sibling form", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:`+refSiblingDescriptionNotDuplicateName+`
    Pet:
      type: object
      properties:
        name:
          $ref: "#/components/schemas/Name"
          description: The name of the pet.
`), nil, "$.components.schemas.Pet.properties.name.description")
	})

	t.Run("accepts an overriding description", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:`+refSiblingDescriptionNotDuplicateName+`
    Pet:
      type: object
      properties:
        name:
          description: What the owner calls the pet.
          allOf:
            - $ref: "#/components/schemas/Name"
`), nil)
	})

	t.Run("ignores targets without a description", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Plain:
      type: string
    Pet:
      type: object
      properties:
        a:
          description: x
          allOf:
            - $ref: "#/components/schemas/Plain"
`), nil)
	})
}
