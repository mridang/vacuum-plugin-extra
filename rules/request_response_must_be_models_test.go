package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequestResponseMustBeModels(t *testing.T) {
	t.Parallel()

	rule := rules.RequestResponseMustBeModels()

	t.Run("flags object bodies without properties", func(t *testing.T) {
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
                additionalProperties:
                  type: string
    post:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
      responses:
        "201":
          description: created
          content:
            application/json; charset=utf-8:
              schema:
                additionalProperties: true
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema",
			"$.paths['/pets'].post.requestBody.content['application/merge-patch+json'].schema",
			"$.paths['/pets'].post.responses['201'].content['application/json; charset=utf-8'].schema",
		}, ruletest.Paths(results))
	})

	t.Run("does not report non object bodies, other rules cover them", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
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
                  type: string
        "201":
          description: ok
          content:
            application/json:
              schema:
                type: string
`), nil)
	})

	t.Run("accepts models, referenced models and compositions", func(t *testing.T) {
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
                $ref: "#/components/schemas/Pet"
        "201":
          description: ok
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/Pet"
                  - type: object
        "202":
          description: ok
          content:
            application/json:
              schema:
                oneOf:
                  - $ref: "#/components/schemas/Pet"
                  - $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
`), nil)
	})

	t.Run("ignores non json media types and missing schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            text/plain:
              schema:
                type: object
            application/vnd.api+json:
              schema:
                type: object
            application/json: {}
        "204":
          description: none
`), nil)
	})

	t.Run("flags a oneOf of dictionaries and untyped property-less composition", func(t *testing.T) {
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
                  - type: object
                    additionalProperties: {type: string}
                  - type: object
                    properties:
                      a:
                        type: string
`), nil, "$.paths['/pets'].get.responses['200'].content['application/json'].schema")
	})
}
