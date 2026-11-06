package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNullableBodySchema(t *testing.T) {
	t.Parallel()

	rule := rules.NullableBodySchema()

	t.Run("flags a nullable body schema", func(t *testing.T) {
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
              nullable: true
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].post.requestBody.content['application/json'].schema", results[0].Path)
	})

	t.Run("flags a nullable schema reached through a ref", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    put:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      nullable: true
`), nil, "$.paths['/pets'].put.requestBody.content['application/json'].schema")
	})

	t.Run("flags a 3.1 type list with null in a component body", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  requestBodies:
    Pet:
      content:
        application/json:
          schema:
            type: [object, "null"]
`), nil, "$.components.requestBodies.Pet.content['application/json'].schema")
	})

	t.Run("accepts non nullable bodies and nullable members", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
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
                  nullable: true
      responses:
        "200":
          description: ok
`), nil)
	})
}
