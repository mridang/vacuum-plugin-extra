package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequestBodyRequiredList(t *testing.T) {
	t.Parallel()

	rule := rules.RequestBodyRequiredList()

	t.Run("flags a body schema without a required list", func(t *testing.T) {
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
                name: {type: string}
      responses:
        "201":
          description: created
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].post.requestBody.content['application/json'].schema", results[0].Path)
		require.Contains(t, results[0].Message, "required")
	})

	t.Run("flags an empty required list and follows refs", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NewPet"
      responses:
        "201":
          description: created
components:
  schemas:
    NewPet:
      type: object
      required: []
      properties:
        name: {type: string}
`), nil, "$.paths['/pets'].post.requestBody.content['application/json'].schema")
	})

	t.Run("accepts a required list and schemas without properties", func(t *testing.T) {
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
              required: [name]
              properties:
                name: {type: string}
      responses:
        "201":
          description: created
    put:
      requestBody:
        content:
          application/json:
            schema:
              type: array
              items:
                type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the mediaTypes option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/xml:
            schema:
              type: object
              properties:
                name: {type: string}
      responses:
        "201":
          description: created
`)

		ruletest.RequireClean(t, rule, document, nil)
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"mediaTypes": []string{"application/xml"}},
			"$.paths['/pets'].post.requestBody.content['application/xml'].schema")
	})
}
