package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSSRFParameterName(t *testing.T) {
	t.Parallel()

	rule := rules.SSRFParameterName()

	t.Run("flags url like names on path items and operations", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /fetch:
    parameters:
      - name: redirect
        in: query
        schema:
          type: string
    get:
      parameters:
        - name: callback_url
          in: query
          schema:
            type: string
        - name: imageUrl
          in: query
          schema:
            type: string
        - name: webhook-url
          in: query
          schema:
            type: string
        - name: callback
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/fetch'].parameters[0].name",
			"$.paths['/fetch'].get.parameters[0].name",
			"$.paths['/fetch'].get.parameters[1].name",
			"$.paths['/fetch'].get.parameters[2].name",
			"$.paths['/fetch'].get.parameters[3].name",
		}, ruletest.Paths(results))
	})

	t.Run("accepts names the original pattern does not match", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
        - name: callbackHandler
          in: query
          schema:
            type: string
        - name: imageURL
          in: query
          schema:
            type: string
        - name: url
          in: query
          schema:
            type: string
        - name: redirect_to
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("does not inspect components or references", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Return"
      responses:
        "200":
          description: ok
components:
  parameters:
    Return:
      name: return_url
      in: query
      schema:
        type: string
`), nil)
	})

	t.Run("honours the pattern option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"pattern": "^limit$"},
			"$.paths['/pets'].get.parameters[0].name")
		ruletest.RequireClean(t, rule, document, ruletest.Options{"pattern": ""})
	})
}
