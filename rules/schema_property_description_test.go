package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSchemaPropertyDescription(t *testing.T) {
	t.Parallel()

	rule := rules.SchemaPropertyDescription()

	t.Run("flags missing and blank descriptions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Owner:
      type: string
      description: Who owns it.
    Pet:
      type: object
      properties:
        name: {type: string}
        tag: {type: string, description: "  "}
        owner: {$ref: "#/components/schemas/Owner"}
        age: {type: integer, description: Age in years.}
`), nil, "$.components.schemas.Pet.properties.name", "$.components.schemas.Pet.properties.tag")
	})

	t.Run("accepts described properties including nested ones", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string, description: The name.}
`), nil)
	})

	t.Run("skips properties under not, if, then and else", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Step:
      type: object
      properties:
        name: {type: string}
      not:
        properties:
          hidden: {type: string}
      if:
        properties:
          gated: {type: string}
      then:
        properties:
          fromThen: {type: string}
      else:
        properties:
          fromElse: {type: string}
`), nil, "$.components.schemas.Step.properties.name")
	})

	t.Run("checks nested inline schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                nested: {type: string}
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].post.requestBody.content['application/json'].schema.properties.nested")
	})
}
