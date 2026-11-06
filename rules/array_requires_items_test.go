package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestArrayRequiresItems(t *testing.T) {
	t.Parallel()

	rule := rules.ArrayRequiresItems()

	t.Run("flags an array without items", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Tags:
      type: array
    Pet:
      type: object
      properties:
        names:
          type: array
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Tags",
			"$.components.schemas.Pet.properties.names",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "items")
	})

	t.Run("ignores constraint fragments below not", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Step:
      type: object
      not:
        properties:
          tags: {type: array, minItems: 1}
`), nil)
	})

	t.Run("accepts an array with items", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Tags:
      type: array
      items:
        type: string
`), nil)
	})

	t.Run("flags a 3.1 type list containing array", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Tags:
      type: [array, "null"]
`), nil, "$.components.schemas.Tags")
	})

	t.Run("accepts prefixItems by default and rejects it when disabled", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
paths: {}
components:
  schemas:
    Pair:
      type: array
      prefixItems:
        - type: string
`)

		ruletest.RequireClean(t, rule, document, nil)
		ruletest.RequirePaths(
			t,
			rule,
			document,
			ruletest.Options{"allowPrefixItems": false},
			"$.components.schemas.Pair",
		)
	})

	t.Run("ignores references and non-array schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Alias:
      $ref: "#/components/schemas/Name"
    Name:
      type: string
`), nil)
	})
}
