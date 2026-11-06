package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSchemaTypeKeywordMismatch(t *testing.T) {
	t.Parallel()

	rule := rules.SchemaTypeKeywordMismatch()

	t.Run("flags contradicting and missing types", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tag:
      type: string
      items:
        type: string
    Pet:
      properties:
        name: {type: string}
    Count:
      type: integer
      properties:
        x: {type: string}
    List:
      items:
        type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Tag.items",
			"$.components.schemas.Pet.properties",
			"$.components.schemas.Count.properties",
			"$.components.schemas.List.items",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results), "schema of type string must not carry items")
		require.Contains(t, ruletest.Messages(results), "schema with properties must declare type: object")
	})

	t.Run("accepts consistent schemas including type arrays", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Tag:
      type: array
      items:
        type: string
    Pet:
      type: [object, "null"]
      properties:
        name: {type: string}
`), nil)
	})

	t.Run("requireExplicitType false only reports contradictions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Tag:
      type: string
      items:
        type: string
    Pet:
      properties:
        name: {type: string}
`), ruletest.Options{"requireExplicitType": false}, "$.components.schemas.Tag.items")
	})

	t.Run("checks inline schemas of operations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
                properties:
                  x: {type: string}
                items:
                  type: string
`), nil, "$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties")
	})
}
