package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const noInlineSchemaDoc = `
paths:
  /pets/{id}:
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
    post:
      parameters:
        - name: filter
          in: query
          schema:
            $ref: "#/components/schemas/Filter"
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
components:
  schemas:
    Filter:
      type: string
`

func TestNoInlineSchema(t *testing.T) {
	t.Parallel()

	rule := rules.NoInlineSchema()

	t.Run("flags inline schemas in all three locations", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(noInlineSchemaDoc), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets/{id}'].parameters[0].schema",
			"$.paths['/pets/{id}'].post.requestBody.content['application/json'].schema",
			"$.paths['/pets/{id}'].post.responses['200'].content['application/json'].schema",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"parameter schema should be moved to components/schemas and referenced",
		)
	})

	t.Run("accepts referenced schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
`), nil)
	})

	t.Run("options switch locations off", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(noInlineSchemaDoc),
			ruletest.Options{"parameters": false, "responses": false},
			"$.paths['/pets/{id}'].post.requestBody.content['application/json'].schema")
	})

	t.Run("allowPrimitives accepts scalar schemas only", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(noInlineSchemaDoc),
			ruletest.Options{"allowPrimitives": true},
			"$.paths['/pets/{id}'].post.requestBody.content['application/json'].schema",
			"$.paths['/pets/{id}'].post.responses['200'].content['application/json'].schema")
	})

	t.Run("checks parameter content schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          content:
            application/json:
              schema:
                type: object
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].get.parameters[0].content['application/json'].schema")
	})
}
