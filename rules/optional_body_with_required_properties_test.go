package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOptionalBodyWithRequiredProperties(t *testing.T) {
	t.Parallel()

	rule := rules.OptionalBodyWithRequiredProperties()

	t.Run("flags an optional body with required properties", func(t *testing.T) {
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
              required: [name]
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].post.requestBody", results[0].Path)
	})

	t.Run("follows refs and allOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        required: false
        content:
          application/json:
            schema:
              allOf:
                - $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      required: [name]
`), nil, "$.paths['/pets'].post.requestBody")
	})

	t.Run("checks component request bodies once", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        $ref: "#/components/requestBodies/Pet"
      responses:
        "200":
          description: ok
components:
  requestBodies:
    Pet:
      content:
        application/json:
          schema:
            type: object
            required: [name]
`), nil, "$.components.requestBodies.Pet")
	})

	t.Run("accepts required bodies and optional properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
    put:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
`), nil)
	})
}
