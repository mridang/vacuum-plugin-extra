package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWellDefinedDictionaries(t *testing.T) {
	t.Parallel()

	rule := rules.WellDefinedDictionaries()

	t.Run("flags a schema that is both model and dictionary", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Labels:
      type: object
      properties:
        name: {type: string}
      additionalProperties:
        type: string
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Labels", results[0].Path)
		require.Contains(t, results[0].Message, "model or a dictionary")
	})

	t.Run("accepts plain models and typed dictionaries", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Model:
      type: object
      properties:
        name: {type: string}
      additionalProperties: false
    Map:
      type: object
      additionalProperties:
        $ref: "#/components/schemas/Model"
    Pattern:
      type: object
      patternProperties:
        "^x-":
          type: string
`), nil)
	})

	t.Run("flags untyped dictionary values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Any:
      type: object
      additionalProperties: true
    Empty:
      type: object
      additionalProperties: {}
    Pattern:
      type: object
      patternProperties:
        "^x-": {}
`), nil,
			"$.components.schemas.Any",
			"$.components.schemas.Empty",
			"$.components.schemas.Pattern")
	})

	t.Run("flags dictionaries of dictionaries", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Nested:
      type: object
      additionalProperties:
        type: object
        additionalProperties:
          type: string
`), nil, "$.components.schemas.Nested")
	})

	t.Run("options relax the value checks", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
components:
  schemas:
    Any:
      type: object
      additionalProperties: true
    Nested:
      type: object
      additionalProperties:
        type: object
        additionalProperties:
          type: string
`)
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"forbidNestedDictionaries": false},
			"$.components.schemas.Any")
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"requireValueType": false},
			"$.components.schemas.Nested")
	})

	t.Run("flags object schemas that define nothing", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
components:
  schemas:
    Free:
      type: object
    Wrapper:
      type: object
      properties:
        inner:
          type: object
    Fine:
      type: object
      properties:
        id: {type: string}
    Scalar:
      type: string
`)
		results := ruletest.Run(t, rule, doc, nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Free",
			"$.components.schemas.Wrapper.properties.inner",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "must define either properties")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"requireObjectDefinition": false})
	})

	t.Run("follows composition like IBM", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Base:
      type: object
      properties:
        id: {type: string}
    Both:
      allOf:
        - $ref: "#/components/schemas/Base"
        - type: object
          additionalProperties:
            type: string
    Untyped:
      type: object
      additionalProperties:
        allOf:
          - type: string
    OneOfTyped:
      type: object
      additionalProperties:
        oneOf:
          - type: string
          - minLength: 1
`), nil,
			"$.components.schemas.Both",
			"$.components.schemas.OneOfTyped",
		)
	})

	t.Run("checks every pattern property value", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Mixed:
      type: object
      patternProperties:
        "^a": {type: string}
        "^b": {description: no type}
`), nil, "$.components.schemas.Mixed")
	})
}
