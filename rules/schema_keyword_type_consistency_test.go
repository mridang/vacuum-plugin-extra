package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSchemaKeywordTypeConsistency(t *testing.T) {
	t.Parallel()

	rule := rules.SchemaKeywordTypeConsistency()

	t.Run("flags inverted bounds", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Age:
      type: integer
      minimum: 10
      maximum: 5
    Name:
      type: string
      minLength: 4
      maxLength: 2
    Tags:
      type: array
      items:
        type: string
      minItems: 3
      maxItems: 1
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Age.maximum",
			"$.components.schemas.Name.maxLength",
			"$.components.schemas.Tags.maxItems",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results), "minimum must not be greater than maximum")
	})

	t.Run("flags keywords that do not apply to the type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Age:
      type: string
      minimum: 0
      multipleOf: 2
    Flag:
      type: boolean
      pattern: "^a$"
    Names:
      type: array
      items:
        type: string
      properties:
        a:
          type: string
`), nil,
			"$.components.schemas.Age.minimum",
			"$.components.schemas.Age.multipleOf",
			"$.components.schemas.Flag.pattern",
			"$.components.schemas.Names.properties",
		)
	})

	t.Run("accepts consistent schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Age:
      type: [integer, "null"]
      minimum: 0
      maximum: 120
    Mixed:
      type: [string, integer]
      minLength: 1
      minimum: 0
    Untyped:
      minimum: 5
      pattern: a
    Pet:
      type: object
      required: [name]
      properties:
        name:
          type: string
`), nil)
	})

	t.Run("flags strict and boolean exclusive bounds", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: integer
      exclusiveMinimum: 5
      maximum: 5
    B:
      type: integer
      minimum: 5
      exclusiveMaximum: 5
    C:
      type: integer
      exclusiveMinimum: 7
      exclusiveMaximum: 3
    D:
      type: array
      items: {type: string}
      contains: {type: string}
      minContains: 3
      maxContains: 1
    E:
      type: integer
      minimum: 5
      maximum: 5
`), nil,
			"$.components.schemas.A.maximum",
			"$.components.schemas.B.exclusiveMaximum",
			"$.components.schemas.C.exclusiveMaximum",
			"$.components.schemas.D.maxContains",
		)

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Empty:
      type: integer
      minimum: 5
      maximum: 5
      exclusiveMinimum: true
    Fine:
      type: integer
      minimum: 5
      maximum: 5
`), nil, "$.components.schemas.Empty.exclusiveMinimum")
	})

	t.Run("flags object and array keywords on other types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    S:
      type: string
      patternProperties:
        "^a": {type: string}
      propertyNames: {pattern: a}
      contains: {type: string}
      prefixItems:
        - type: string
`), nil,
			"$.components.schemas.S.patternProperties",
			"$.components.schemas.S.propertyNames",
			"$.components.schemas.S.contains",
			"$.components.schemas.S.prefixItems",
		)
	})
}
