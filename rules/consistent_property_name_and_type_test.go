package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const consistentPropertyNameAndTypeDoc = `
components:
  schemas:
    Id:
      type: integer
    Pet:
      type: object
      properties:
        id: {type: string}
        name: {type: string}
    Owner:
      type: object
      properties:
        id: {$ref: "#/components/schemas/Id"}
        name: {type: string}
`

func TestConsistentPropertyNameAndType(t *testing.T) {
	t.Parallel()

	rule := rules.ConsistentPropertyNameAndType()

	t.Run("flags later declarations with another type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(consistentPropertyNameAndTypeDoc), nil,
			"$.components.schemas.Owner.properties.id")
	})

	t.Run("accepts matching types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        id: {type: string}
    Owner:
      type: object
      properties:
        id: {type: string}
`), nil)
	})

	t.Run("treats nullable type lists as different in 3.1", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        age: {type: integer}
    Owner:
      type: object
      properties:
        age: {type: [integer, "null"]}
`), nil, "$.components.schemas.Owner.properties.age")
	})

	t.Run("honours the ignore option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(consistentPropertyNameAndTypeDoc),
			ruletest.Options{"ignore": []any{"id"}})
	})
}
