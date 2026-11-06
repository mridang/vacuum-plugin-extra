package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestServerXInternal(t *testing.T) {
	t.Parallel()

	rule := rules.ServerXInternal()

	t.Run("flags servers without the extension", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com
    x-internal: false
  - url: https://staging.example.com
paths:
  /pets:
    servers:
      - url: https://pets.example.com
    get:
      servers:
        - url: https://get.example.com
          x-internal: true
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{"$.servers[1]", "$.paths['/pets'].servers[0]"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "x-internal")
	})

	t.Run("accepts booleans on every server", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
servers:
  - url: https://api.example.com
    x-internal: false
  - url: https://staging.example.com
    x-internal: true
`), nil)
	})

	t.Run("rejects non boolean values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://a.example.com
    x-internal: "true"
  - url: https://b.example.com
    x-internal: yes please
`), nil, "$.servers[0]['x-internal']", "$.servers[1]['x-internal']")
	})

	t.Run("honours the extension option and absent servers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
`), nil)
		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://a.example.com
    x-internal: true
`), ruletest.Options{"extension": "x-private"}, "$.servers[0]")
	})
}
