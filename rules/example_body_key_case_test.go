package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestExampleBodyKeyCase(t *testing.T) {
	t.Parallel()

	rule := rules.ExampleBodyKeyCase()

	t.Run("flags camel case keys at every depth in a snake case API", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
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
                  firstName: {type: string}
                  owner_id: {type: integer}
                  address:
                    type: object
                    properties:
                      zipCode: {type: string}
                  tags:
                    type: array
                    items:
                      type: object
                      properties:
                        lastSeen: {type: string}
              example:
                firstName: Ada
                owner_id: 1
                address:
                  zipCode: "123"
                tags:
                  - lastSeen: now
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.firstName",
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.address.zipCode",
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.tags[0].lastSeen",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"example body property 'firstName' must be snake_case (for example 'first_name')",
		)
	})

	t.Run("accepts snake case examples", func(t *testing.T) {
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
                first_name: {type: string}
                id2: {type: integer}
            examples:
              basic:
                value:
                  first_name: Ada
                  id2: 1
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("checks named examples and honours the casing option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                first_name: {type: string}
                lastName: {type: string}
            examples:
              basic:
                value:
                  first_name: Ada
                  lastName: Lovelace
      responses:
        "200":
          description: ok
`), ruletest.Options{"casing": "camelCase"},
			"$.paths['/pets'].post.requestBody.content['application/json'].examples.basic.value.first_name")
	})

	t.Run("supports kebab case and acronyms", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  responses:
    Ok:
      description: ok
      content:
        application/json:
          schema:
            type: object
            properties:
              user-id: {type: integer}
              userID: {type: integer}
          example:
            user-id: 1
            userID: 2
`), ruletest.Options{"casing": "kebab-case"}, "$.components.responses.Ok.content['application/json'].example.userID")
	})

	t.Run("ignores non json media types", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/xml:
              example:
                firstName: Ada
`)
		ruletest.RequireClean(t, rule, doc, nil)
	})

	t.Run("does not check keys of free-form maps, embedded documents or schemaless examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(exampleBodyKeyCaseFreeForm), nil,
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.camelName",
		)
	})

	t.Run("follows $ref, allOf and additionalProperties into declared objects", func(t *testing.T) {
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
                $ref: "#/components/schemas/Pet"
              example:
                firstName: Ada
                labels:
                  SOME_KEY:
                    innerName: x
                    inner_ok: y
components:
  schemas:
    Base:
      type: object
      properties:
        firstName: {type: string}
    Pet:
      allOf:
        - $ref: "#/components/schemas/Base"
        - type: object
          properties:
            labels:
              type: object
              additionalProperties:
                type: object
                properties:
                  innerName: {type: string}
                  inner_ok: {type: string}
`), nil,
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.firstName",
			"$.paths['/pets'].get.responses['200'].content['application/json'].example.labels.SOME_KEY.innerName",
		)
	})

	t.Run("ignoreKeys skips exact keys and glob patterns", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(exampleBodyKeyCaseClaims)
		prefix := "$.paths['/me'].get.responses['200'].content['application/json'].example."

		ruletest.RequirePaths(t, rule, document, nil,
			prefix+"givenName", prefix+"familyName", prefix+"zipCode")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"ignoreKeys": []string{"givenName", "familyName"}},
			prefix+"zipCode")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"ignoreKeys": "*Name"}, prefix+"zipCode")
		ruletest.RequireClean(t, rule, document, ruletest.Options{"ignoreKeys": []any{"*Name", "zip?ode"}})
	})

	t.Run("a malformed ignoreKeys pattern is reported", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(exampleBodyKeyCaseClaims), ruletest.Options{"ignoreKeys": "[a"})

		require.Equal(t, []string{"$"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, `option ignoreKeys: invalid pattern "[a"`)
	})
}

const exampleBodyKeyCaseFreeForm = `
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
                  camelName: {type: string}
                  variables:
                    type: object
                    additionalProperties: {type: string}
                  file:
                    type: object
                  content:
                    type: string
              example:
                camelName: x
                variables:
                  GOOGLE_CLIENT_ID: abc
                file:
                  someKey: 1
                  nested:
                    otherKey: 2
                content: '{"aB": 1}'
        "201":
          description: created
          content:
            application/json:
              example:
                noSchemaKey: 1
`

const exampleBodyKeyCaseClaims = `
paths:
  /me:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  givenName: {type: string}
                  familyName: {type: string}
                  zipCode: {type: string}
                  zip_code: {type: string}
              example:
                givenName: Ada
                familyName: Lovelace
                zipCode: "1"
                zip_code: "1"
`
