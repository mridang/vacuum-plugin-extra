package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestIntegerBoundsInFormatRange(t *testing.T) {
	t.Parallel()

	rule := rules.IntegerBoundsInFormatRange()

	t.Run("flags missing format, missing bounds and out-of-range bounds", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        a: {type: integer, minimum: 0, maximum: 5}
        b: {type: integer, format: int32, minimum: 0}
        c: {type: integer, format: int32, minimum: 0, maximum: 5000000000}
        d: {type: integer, format: int64, minimum: 0, maximum: 9223372036854775807}
        e: {type: integer, format: int64, minimum: 0, maximum: 9223372036854775808}
`), nil,
			"$.components.schemas.Pet.properties.a",
			"$.components.schemas.Pet.properties.b",
			"$.components.schemas.Pet.properties.c.maximum",
			"$.components.schemas.Pet.properties.e.maximum")
	})

	t.Run("accepts a fully specified integer", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age: {type: integer, format: int32, minimum: 0, maximum: 100}
        name: {type: string}
`), nil)
	})

	t.Run("options relax the requirements", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age: {type: [integer, "null"]}
`), ruletest.Options{"requireFormat": false, "requireBounds": false})
	})
}
