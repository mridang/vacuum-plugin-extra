package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRefPatternComponentLocation(t *testing.T) {
	t.Parallel()

	rule := rules.RefPatternComponentLocation()

	t.Run("flags references into the wrong section or path", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/parameters/Owner"
        "404":
          $ref: "#/components/responses/NotFound"
components:
  schemas:
    Pet:
      type: object
      properties:
        owner:
          $ref: "#/components/parameters/Owner"
        ok:
          type: string
  parameters:
    Owner:
      name: owner
      in: query
      schema:
        type: string
  responses:
    NotFound:
      description: missing
`), nil,
			"$.paths['/pets'].get.parameters[0]['$ref']",
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema['$ref']",
			"$.components.schemas.Pet.properties.owner['$ref']")
	})

	t.Run("accepts correct references", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        $ref: "#/components/requestBodies/Body"
      parameters:
        - $ref: "#/components/parameters/Limit"
      responses:
        "200":
          description: ok
components:
  requestBodies:
    Body:
      content:
        application/json:
          schema:
            allOf:
              - $ref: "#/components/schemas/Pet"
  parameters:
    Limit:
      name: limit
      in: query
      schema:
        type: integer
  schemas:
    Pet:
      type: object
`), nil)
	})

	t.Run("checks examples, headers, links, callbacks, security schemes and nested schemas", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      requestBody:
        $ref: "#/components/schemas/Body"
      responses:
        "200":
          description: ok
          headers:
            X-Rate:
              $ref: "#/components/schemas/Header"
          links:
            next:
              $ref: "#/components/schemas/Link"
          content:
            application/json:
              examples:
                one:
                  $ref: "#/components/schemas/Example"
              schema:
                type: array
                items:
                  $ref: "#/components/parameters/Item"
                additionalProperties:
                  $ref: "#/components/parameters/Extra"
      callbacks:
        cb:
          $ref: "#/components/schemas/Callback"
components:
  schemas:
    Body: {type: object}
    Header: {type: object}
    Link: {type: object}
    Example: {type: object}
    Callback: {type: object}
    Key: {type: object}
  parameters:
    Item: {name: item, in: query, schema: {type: string}}
    Extra: {name: extra, in: query, schema: {type: string}}
  securitySchemes:
    Key:
      $ref: "#/components/schemas/Key"
`), nil)

		require.Equal(t, []string{
			"$.paths['/pets'].get.requestBody['$ref']",
			"$.paths['/pets'].get.responses['200'].headers['X-Rate']['$ref']",
			"$.paths['/pets'].get.responses['200'].links.next['$ref']",
			"$.paths['/pets'].get.responses['200'].content['application/json'].examples.one['$ref']",
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.items['$ref']",
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.additionalProperties['$ref']",
			"$.paths['/pets'].get.callbacks.cb['$ref']",
			"$.components.securitySchemes.Key['$ref']",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"$refs to requestBodies should start with '#/components/requestBodies/'",
		)
	})

	t.Run("accepts nested pointers and non-string $ref properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        a:
          $ref: "#/components/schemas/Other/properties/name"
        $ref:
          type: string
    Other:
      type: object
      properties:
        name: {type: string}
`), nil)
	})
}
