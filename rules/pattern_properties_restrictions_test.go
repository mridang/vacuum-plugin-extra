package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPatternPropertiesRestrictions(t *testing.T) {
	t.Parallel()

	rule := rules.PatternPropertiesRestrictions()

	t.Run("flags each restriction", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Both:
      type: object
      additionalProperties: false
      patternProperties:
        "^a$":
          type: string
    Many:
      type: object
      patternProperties:
        "^a$":
          type: string
        "^b$":
          type: string
    Empty:
      type: object
      patternProperties: {}
    Loose:
      type: object
      patternProperties:
        "a":
          type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Both",
			"$.components.schemas.Many.patternProperties",
			"$.components.schemas.Empty.patternProperties",
			"$.components.schemas.Loose.patternProperties.a",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "patternProperties")
	})

	t.Run("accepts a single anchored pattern", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths: {}
components:
  schemas:
    Labels:
      type: object
      patternProperties:
        "^x-.*$":
          type: string
`), nil)
	})

	t.Run("only applies to OpenAPI 3.1 documents", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Many:
      type: object
      patternProperties:
        "^a$":
          type: string
        "^b$":
          type: string
`), nil)
	})
}
