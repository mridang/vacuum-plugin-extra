package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWireParameterNoCamelCase(t *testing.T) {
	t.Parallel()

	rule := rules.WireParameterNoCamelCase()

	t.Run("flags camel case parameter names everywhere", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: ownerId
        in: query
        schema:
          type: string
    get:
      parameters:
        - name: pageSize
          in: query
          schema:
            type: integer
        - $ref: "#/components/parameters/SortBy"
      responses:
        "200":
          description: ok
components:
  parameters:
    SortBy:
      name: sortBy
      in: query
      schema:
        type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0].name",
			"$.paths['/pets'].get.parameters[0].name",
			"$.components.parameters.SortBy.name",
		}, ruletest.Paths(results))
	})

	t.Run("accepts kebab, snake and capitalised names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: Idempotency-Key
          in: header
          schema:
            type: string
        - name: Origin
          in: header
          schema:
            type: string
        - name: page_size
          in: query
          schema:
            type: integer
        - name: _zflow_request
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the pattern option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: pageSize
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"pattern": ""})
		ruletest.RequireClean(t, rule, document, ruletest.Options{"pattern": "^zzz$"})
	})
}
