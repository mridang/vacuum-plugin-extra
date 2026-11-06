package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPutBodyMatchesGetResponse(t *testing.T) {
	t.Parallel()

	rule := rules.PutBodyMatchesGetResponse()

	t.Run("flags a put body that differs from the get response", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    put:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NewPet"
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string}
        id: {type: integer}
    NewPet:
      type: object
      properties:
        name: {type: string}
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets/{id}'].put.requestBody.content['application/json'].schema", results[0].Path)
		require.Contains(t, results[0].Message, "200")
	})

	t.Run("accepts the same reference, and an inlined copy of it", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    put:
      requestBody:
        content:
          application/json:
            schema:
              description: inline copy
              type: object
              properties:
                name: {description: the name, type: string}
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string}
`), nil)
	})

	t.Run("ignores paths without both operations and unshared media types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /a:
    put:
      requestBody:
        content:
          application/json:
            schema: {type: string}
      responses:
        "200":
          description: ok
  /b:
    get:
      responses:
        "200":
          description: ok
          content:
            application/xml:
              schema: {type: string}
    put:
      requestBody:
        content:
          application/json:
            schema: {type: integer}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the responseCode option", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema: {type: string}
        "206":
          description: partial
          content:
            application/json:
              schema: {type: integer}
    put:
      requestBody:
        content:
          application/json:
            schema: {type: integer}
      responses:
        "200":
          description: ok
`)
		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].put.requestBody.content['application/json'].schema")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"responseCode": "206"})
	})
}
