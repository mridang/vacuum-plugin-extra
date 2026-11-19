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

	t.Run("treats a nullable type list as its plain type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        purpose: {type: string}
    Owner:
      type: object
      properties:
        purpose: {type: [string, "null"]}
`), nil)
	})

	t.Run("does not give a property whose only type is null a type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        purpose: {type: string}
    Owner:
      type: object
      properties:
        purpose: {type: "null"}
`), nil)
	})

	t.Run("reports null as a type when ignoreNull is false", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        purpose: {type: string}
    Owner:
      type: object
      properties:
        purpose: {type: [string, "null"]}
`), ruletest.Options{"ignoreNull": false}, "$.components.schemas.Owner.properties.purpose")
	})

	t.Run("still reports a string against an object", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        purpose: {type: string}
    Owner:
      type: object
      properties:
        purpose: {type: object}
`), nil, "$.components.schemas.Owner.properties.purpose")
	})

	t.Run("honours the ignore option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(consistentPropertyNameAndTypeDoc),
			ruletest.Options{"ignore": []any{"id"}})
	})
}
