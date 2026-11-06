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
