package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPropertyNameCollisionAfterNormalising(t *testing.T) {
	t.Parallel()

	rule := rules.PropertyNameCollisionAfterNormalising()

	t.Run("flags properties that collide", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Thing:
      type: object
      properties:
        thingType: {type: string}
        thing_type: {type: string}
        Thing-Type: {type: string}
        other: {type: string}
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Thing.properties.thing_type",
			"$.components.schemas.Thing.properties['Thing-Type']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "thingType")
	})

	t.Run("accepts distinct names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Thing:
      type: object
      properties:
        thingType: {type: string}
        thingKind: {type: string}
`), nil)
	})

	t.Run("does not compare across schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: object
      properties:
        fooBar: {type: string}
    B:
      type: object
      properties:
        foo_bar: {type: string}
`), nil)
	})
}
