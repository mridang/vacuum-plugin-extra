package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestFormatMatchesType(t *testing.T) {
	t.Parallel()

	rule := rules.FormatMatchesType()

	t.Run("flags mismatching formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age: {type: string, format: int32}
        born: {type: integer, format: date-time}
        weight: {type: number, format: double}
        ok: {type: integer, format: int64}
`), nil,
			"$.components.schemas.Pet.properties.age.format",
			"$.components.schemas.Pet.properties.born.format")
	})

	t.Run("accepts matching and unknown formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        id: {type: string, format: uuid}
        custom: {type: integer, format: my-format}
        untyped: {format: int32}
`), nil)
	})

	t.Run("accepts any matching type of a 3.1 type list", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age: {type: [integer, "null"], format: int32}
        bad: {type: [string, "null"], format: int32}
`)
		ruletest.RequirePaths(t, rule, doc, nil, "$.components.schemas.Pet.properties.bad.format")
	})
}
