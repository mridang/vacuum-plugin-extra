package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const responseRequiredPropertiesDoc = `
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingCollection"
components:
  schemas:
    ThingCollection:
      type: object
      properties:
        things:
          type: array
          items:
            $ref: "#/components/schemas/Thing"
        status:
          type: string
          enum: [on, off]
    Thing:
      type: object
      required: [id]
      properties:
        id:
          type: string
        tags:
          type: array
          items:
            type: string
`

func TestResponseRequiredProperties(t *testing.T) {
	t.Parallel()

	rule := rules.ResponseRequiredProperties()

	t.Run("flags optional array and enum properties through refs", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(responseRequiredPropertiesDoc), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.ThingCollection.properties.things",
			"$.components.schemas.ThingCollection.properties.status",
			"$.components.schemas.Thing.properties.tags",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"array property 'things' in a response schema must be listed in required",
		)
	})

	t.Run("accepts required properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [things, status]
                properties:
                  things:
                    type: array
                    items: {type: string}
                  status:
                    type: string
                    enum: [on, off]
`), nil)
	})

	t.Run("options select the property kinds", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc31(responseRequiredPropertiesDoc)
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"enums": false},
			"$.components.schemas.ThingCollection.properties.things",
			"$.components.schemas.Thing.properties.tags")
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"arrays": false},
			"$.components.schemas.ThingCollection.properties.status")
	})

	t.Run("honours requirements from sibling allOf members", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                allOf:
                  - type: object
                    properties:
                      things:
                        type: array
                        items: {type: string}
                  - type: object
                    required: [things]
`), nil)
	})

	t.Run("ignores schemas that are not used by responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Unused"
      responses:
        "204":
          description: none
components:
  schemas:
    Unused:
      type: object
      properties:
        list:
          type: array
          items: {type: string}
`), nil)
	})
}
