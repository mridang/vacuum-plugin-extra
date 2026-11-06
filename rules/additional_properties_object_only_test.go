package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAdditionalPropertiesObjectOnly(t *testing.T) {
	t.Parallel()

	rule := rules.AdditionalPropertiesObjectOnly()

	t.Run("flags additionalProperties on a string schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Name:
      type: string
      additionalProperties: false
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Name.additionalProperties", results[0].Path)
		require.Contains(t, results[0].Message, "string")
	})

	t.Run("flags nested array schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        tags:
          type: array
          items:
            type: string
          additionalProperties: true
`), nil, "$.components.schemas.Pet.properties.tags.additionalProperties")
	})

	t.Run("accepts object schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Map:
      type: object
      additionalProperties:
        type: string
`), nil)
	})

	t.Run("ignores schemas without a type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Any:
      additionalProperties: true
`), nil)
	})

	t.Run("accepts a 3.1 type list that includes object", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Maybe:
      type: [object, "null"]
      additionalProperties: false
`), nil)
	})

	t.Run("flags a 3.1 type list without object", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Maybe:
      type: [string, "null"]
      additionalProperties: false
`), nil, "$.components.schemas.Maybe.additionalProperties")
	})
}
