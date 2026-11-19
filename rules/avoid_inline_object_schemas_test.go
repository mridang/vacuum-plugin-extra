package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAvoidInlineObjectSchemas(t *testing.T) {
	t.Parallel()

	rule := rules.AvoidInlineObjectSchemas()

	t.Run("accepts unions of references and null when allowed", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - $ref: "#/components/schemas/Cat"
                  - $ref: "#/components/schemas/Dog"
components:
  schemas:
    Cat: {type: object, properties: {a: {type: string}}}
    Dog: {type: object, properties: {b: {type: string}}}
    Owner:
      type: object
      properties:
        pet:
          anyOf:
            - $ref: "#/components/schemas/Cat"
            - type: "null"
`), ruletest.Options{"allowReferenceUnions": true})
	})

	t.Run("accepts an inline allOf member next to a reference when allowed", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(avoidInlineAllOfExtension), ruletest.Options{"allowAllOfExtension": true})
	})

	t.Run("flags an inline allOf member next to a reference by default", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(avoidInlineAllOfExtension), nil)

		require.Equal(t, []string{"$.components.schemas.Extended.allOf[1]"}, ruletest.Paths(results))
	})

	t.Run("flags an allOf of inline members only when extension is allowed", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Both:
      allOf:
        - type: object
          properties: {a: {type: string}}
        - type: object
          properties: {b: {type: string}}
`), ruletest.Options{"allowAllOfExtension": true})

		require.Equal(t,
			[]string{"$.components.schemas.Both.allOf[0]", "$.components.schemas.Both.allOf[1]"},
			ruletest.Paths(results),
		)
	})

	t.Run("flags an inline request body object", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        "201":
          description: created
`), nil)

		require.Equal(t,
			[]string{"$.paths['/pets'].post.requestBody.content['application/json'].schema"},
			ruletest.Paths(results),
		)
	})

	t.Run("flags an inline object nested in a component and in a response", func(t *testing.T) {
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
                items:
                  type: object
                  properties:
                    id:
                      type: string
components:
  schemas:
    Owner:
      type: object
      properties:
        address:
          type: object
          properties:
            street:
              type: string
`), nil,
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.items",
			"$.components.schemas.Owner.properties.address",
		)
	})

	t.Run("flags an inline composition", func(t *testing.T) {
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
                oneOf:
                  - $ref: "#/components/schemas/Cat"
components:
  schemas:
    Cat:
      type: object
      properties:
        name:
          type: string
`), nil, "$.paths['/pets'].get.responses['200'].content['application/json'].schema")
	})

	t.Run("accepts referenced and primitive schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
`), nil)
	})

	t.Run("skips an inline object under not but flags the same object elsewhere", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                not:
                  type: object
                  properties:
                    id:
                      type: string
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`), nil, "$.paths['/pets'].get.responses['201'].content['application/json'].schema")
	})

	t.Run("accepts a free form object", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
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
`), nil)
	})
}

const avoidInlineAllOfExtension = `
components:
  schemas:
    Base: {type: object, properties: {a: {type: string}}}
    Extended:
      allOf:
        - $ref: "#/components/schemas/Base"
        - type: object
          properties: {b: {type: string}}
`
