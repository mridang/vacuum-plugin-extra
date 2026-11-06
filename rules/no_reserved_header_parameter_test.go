package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoReservedHeaderParameter(t *testing.T) {
	t.Parallel()

	rule := rules.NoReservedHeaderParameter()

	t.Run("flags each default reserved header", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: accept, in: header, schema: {type: string}}
        - {name: Authorization, in: header, schema: {type: string}}
        - {name: Content-Type, in: header, schema: {type: string}}
        - {name: If-Modified-Since, in: header, schema: {type: string}}
        - {name: IF-UNMODIFIED-SINCE, in: header, schema: {type: string}}
        - {name: X-Trace, in: header, schema: {type: string}}
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.parameters[0]",
			"$.paths['/pets'].get.parameters[1]",
			"$.paths['/pets'].get.parameters[2]",
			"$.paths['/pets'].get.parameters[3]",
			"$.paths['/pets'].get.parameters[4]",
		}, ruletest.Paths(results))
	})

	t.Run("ignores query parameters with a reserved name", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: Accept, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags path level and component parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    parameters:
      - {name: Accept, in: header, schema: {type: string}}
    get:
      responses:
        "200":
          description: ok
components:
  parameters:
    Auth: {name: Authorization, in: header, schema: {type: string}}
`), nil, "$.paths['/pets'].parameters[0]", "$.components.parameters.Auth")
	})

	t.Run("honours the headers option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: Accept, in: header, schema: {type: string}}
        - {name: X-Trace, in: header, schema: {type: string}}
      responses:
        "200":
          description: ok
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"headers": []string{"x-trace"}},
			"$.paths['/pets'].get.parameters[1]")
	})
}
