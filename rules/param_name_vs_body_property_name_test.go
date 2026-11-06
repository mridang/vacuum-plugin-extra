package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestParamNameVsBodyPropertyName(t *testing.T) {
	t.Parallel()

	rule := rules.ParamNameVsBodyPropertyName()

	t.Run("flags a parameter named like a body property", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
    put:
      parameters:
        - {name: name, in: query, schema: {type: string}}
        - {name: limit, in: query, schema: {type: integer}}
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: {type: string}
                name: {type: string}
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets/{id}'].put.parameters[0]",
			"$.paths['/pets/{id}'].parameters[0]",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "request body")
	})

	t.Run("follows refs and composition", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    post:
      parameters:
        - {name: color, in: header, schema: {type: string}}
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              allOf:
                - $ref: "#/components/schemas/Base"
      responses:
        "200":
          description: ok
components:
  schemas:
    Base:
      type: object
      properties:
        color: {type: string}
`), nil, "$.paths['/pets'].post.parameters[0]")
	})

	t.Run("ignores read only properties and unrelated names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id: {type: string, readOnly: true}
                name: {type: string}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores non JSON request bodies and missing bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      parameters:
        - {name: name, in: query, schema: {type: string}}
      requestBody:
        content:
          text/plain:
            schema:
              type: object
              properties:
                name: {type: string}
      responses:
        "200":
          description: ok
    get:
      parameters:
        - {name: name, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
`), nil)
	})
}
