package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoNullableProperties(t *testing.T) {
	t.Parallel()

	rule := rules.NoNullableProperties()

	t.Run("flags nullable properties in 3.0", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Name:
      type: string
      nullable: true
    Pet:
      type: object
      properties:
        nickname: {type: string, nullable: true}
        name: {$ref: "#/components/schemas/Name"}
        age: {type: integer}
`), nil,
			"$.components.schemas.Pet.properties.nickname",
			"$.components.schemas.Pet.properties.name")
	})

	t.Run("flags null types in 3.1", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        nickname: {type: [string, "null"]}
`), nil, "$.components.schemas.Pet.properties.nickname")
	})

	t.Run("ignores constraint fragments below not", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Step:
      type: object
      not:
        properties:
          note:
            anyOf:
              - type: "null"
              - type: string
`), nil)
	})

	t.Run("allows nullable properties in merge-patch bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
              properties:
                nickname: {type: string, nullable: true}
      responses:
        "200":
          description: ok
`), nil)
	})
}
