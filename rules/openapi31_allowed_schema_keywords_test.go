package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOpenapi31AllowedSchemaKeywords(t *testing.T) {
	t.Parallel()

	rule := rules.Openapi31AllowedSchemaKeywords()

	t.Run("flags keywords that are not part of OpenAPI 3.1", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Kind:
      type: string
      nullable: true
    List:
      type: array
      itemz:
        type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Kind.nullable",
			"$.components.schemas.List.itemz",
		}, ruletest.Paths(results))
	})

	t.Run("accepts the whole JSON Schema 2020-12 vocabulary by default", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Kind:
      $schema: https://json-schema.org/draft/2020-12/schema
      type: string
      const: cat
      deprecated: true
    List:
      type: array
      prefixItems:
        - type: string
      contains: {type: string}
    Map:
      type: object
      propertyNames: {pattern: "^[a-z]+$"}
`), nil)
	})

	t.Run("flags keywords outside the ibm profile", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Kind:
      type: string
      const: cat
    List:
      type: array
      prefixItems:
        - type: string
`), ruletest.Options{"profile": "ibm"})

		require.ElementsMatch(t, []string{
			"$.components.schemas.Kind.const",
			"$.components.schemas.List.prefixItems",
		}, ruletest.Paths(results))
	})

	t.Run("accepts allowed keywords and extensions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Kind:
      type: string
      enum: [cat]
      x-internal: true
    Pet:
      type: object
      properties:
        const:
          type: string
`), nil)
	})

	t.Run("honours the allowedKeywords option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
components:
  schemas:
    Kind:
      type: string
      const: cat
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"allowedKeywords": []string{"type", "const"}})
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"allowedKeywords": []string{"const"}},
			"$.components.schemas.Kind.type")
	})

	t.Run("does not apply to OpenAPI 3.0 documents", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Kind:
      type: string
      nullable: true
`), nil)
	})
}
