package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWellKnownPropertyTypes(t *testing.T) {
	t.Parallel()

	rule := rules.WellKnownPropertyTypes()

	t.Run("flags wrong types and formats", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        id: {type: integer}
        created: {type: string}
        modified: {type: integer, format: int64}
        name: {type: integer}
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet.properties.id",
			"$.components.schemas.Pet.properties.created",
			"$.components.schemas.Pet.properties.modified",
			"$.components.schemas.Pet.properties.modified",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "must")
	})

	t.Run("accepts the common types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        id: {type: [string, "null"]}
        created: {type: string, format: date-time}
        modified:
          $ref: "#/components/schemas/Timestamp"
    Timestamp:
      type: string
      format: date-time
`), nil)
	})

	t.Run("flags a property without a type", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        id: {description: anything}
`), nil, "$.components.schemas.Pet.properties.id")
	})

	t.Run("checks inline schemas and honours options", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  uuid: {type: integer}
                  id: {type: integer}
`)

		ruletest.RequirePaths(t, rule, document, nil,
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties.id")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"idProperties": []string{"uuid"}},
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties.uuid")
	})
}
