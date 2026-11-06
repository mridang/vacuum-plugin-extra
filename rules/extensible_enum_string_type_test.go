package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestExtensibleEnumStringType(t *testing.T) {
	t.Parallel()

	rule := rules.ExtensibleEnumStringType()

	t.Run("flags non string schemas and properties", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Colour:
      type: integer
      x-extensible-enum: [1, 2]
    Pet:
      type: object
      properties:
        kind:
          x-extensible-enum: [cat, dog]
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Colour",
			"$.components.schemas.Pet.properties.kind",
		}, ruletest.Paths(results))
	})

	t.Run("accepts string schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Colour:
      type: string
      x-extensible-enum: [red, green]
    Plain:
      type: integer
`), nil)
	})

	t.Run("accepts a nullable string type list in 3.1", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Colour:
      type: [string, "null"]
      x-extensible-enum: [red]
`), nil)
	})

	t.Run("flags a type list with another type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Colour:
      type: [string, integer]
      x-extensible-enum: [red]
`), nil, "$.components.schemas.Colour")
	})
}
