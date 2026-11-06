package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSchemaDescriptionRequired(t *testing.T) {
	t.Parallel()

	rule := rules.SchemaDescriptionRequired()

	t.Run("flags schemas and properties without a description", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
          description: The name.
        age:
          type: integer
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet",
			"$.components.schemas.Pet.properties.age",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "description")
	})

	t.Run("accepts described schemas, references and described compositions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Base:
      type: object
      description: The base.
    Pet:
      description: A pet.
      allOf:
        - $ref: "#/components/schemas/Base"
    Alias:
      $ref: "#/components/schemas/Base"
    Wrapper:
      allOf:
        - $ref: "#/components/schemas/Base"
`), nil)
	})

	t.Run("scope components ignores nested schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      description: A pet.
      properties:
        age:
          type: integer
    Owner:
      type: object
`), ruletest.Options{"scope": "components"}, "$.components.schemas.Owner")
	})

	t.Run("treats a blank description as missing", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      description: "  "
`), nil, "$.components.schemas.Pet")
	})
}
