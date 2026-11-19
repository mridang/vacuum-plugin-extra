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

	t.Run("flags a schema without a required key that is reached through a ref", func(t *testing.T) {
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
      properties:
        name: {type: string}
`), nil, "$.paths['/pets'].post.requestBody.content['application/json'].schema")
	})

	t.Run("accepts an explicit empty required list", func(t *testing.T) {
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
              required: []
              properties:
                name: {type: string}
      responses:
        "201":
          description: created
`), nil)
	})

	t.Run("accepts an explicit empty required list reached through a ref", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
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
`), nil)
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

	t.Run("does not count readOnly properties as properties to require", func(t *testing.T) {
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
                id: {type: string, readOnly: true}
                created: {$ref: "#/components/schemas/Stamp"}
      responses:
        "201":
          description: created
components:
  schemas:
    Stamp:
      type: string
      readOnly: true
`), nil)
	})

	t.Run("does not accept a required list that names only readOnly properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [id]
              properties:
                id: {type: string, readOnly: true}
                name: {type: string}
      responses:
        "201":
          description: created
`), nil, "$.paths['/pets'].post.requestBody.content['application/json'].schema")
	})

	t.Run("accepts a required list that names a writable property beside readOnly ones", func(t *testing.T) {
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
                id: {type: string, readOnly: true}
                name: {type: string}
      responses:
        "201":
          description: created
`), nil)
	})
}
