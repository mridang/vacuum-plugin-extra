package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestServerVariablesDefined(t *testing.T) {
	t.Parallel()

	rule := rules.ServerVariablesDefined()

	t.Run("flags undefined and unused variables", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
servers:
  - url: https://{region}.example.com/{version}
    variables:
      version:
        default: v1
      env:
        default: prod
paths: {}
`), nil)

		require.ElementsMatch(t, []string{
			"$.servers[0].url",
			"$.servers[0].variables.env",
		}, ruletest.Paths(results))
	})

	t.Run("accepts matching variables", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://{region}.example.com
    variables:
      region:
        default: eu
  - url: https://static.example.com
paths: {}
`), nil)
	})

	t.Run("checks path item and operation servers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    servers:
      - url: https://{a}.example.com
    get:
      servers:
        - url: https://x.example.com
          variables:
            b:
              default: c
      responses:
        "200":
          description: ok
`), nil,
			"$.paths['/pets'].servers[0].url",
			"$.paths['/pets'].get.servers[0].variables.b",
		)
	})
}
