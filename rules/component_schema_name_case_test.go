package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestComponentSchemaNameCase(t *testing.T) {
	t.Parallel()

	rule := rules.ComponentSchemaNameCase()

	t.Run("flags names that are not PascalCase", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    pet_store:
      type: object
    petStore:
      type: object
    Pet-Owner:
      type: object
    PetStore:
      type: object
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.pet_store",
			"$.components.schemas.petStore",
			"$.components.schemas['Pet-Owner']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "upper camel case")
	})

	t.Run("accepts PascalCase names including acronyms", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
    HTTPError:
      type: object
    ID:
      type: string
`), nil)
	})

	t.Run("upper-camel style applies the IBM expression", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    ID:
      type: string
    Pet:
      type: object
    PetStore2:
      type: object
    pet:
      type: object
    Pet_Store:
      type: object
`), ruletest.Options{"style": "upper-camel"},
			"$.components.schemas.ID",
			"$.components.schemas.pet",
			"$.components.schemas.Pet_Store",
		)
	})

	t.Run("does nothing without component schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
`), nil)
	})
}
