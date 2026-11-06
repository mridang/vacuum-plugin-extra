package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestServerEnvironmentDescription(t *testing.T) {
	t.Parallel()

	rule := rules.ServerEnvironmentDescription()

	t.Run("flags unspecific descriptions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com
    description: The main server
  - url: https://ok.example.com
    description: Production
`), nil, "$.servers[0].description")
	})

	t.Run("ignores servers without description by default", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://other.example.com
`), nil)
	})

	t.Run("reports servers without description when required", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://other.example.com
  - url: https://ok.example.com
    description: Production
`), ruletest.Options{"requireDescription": true}, "$.servers[0]")
	})

	t.Run("accepts the original extra names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://a.example.com
    description: Preproduction
  - url: https://b.example.com
    description: Next release
`), nil)
	})

	t.Run("accepts environment names in any case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
servers:
  - url: https://a.example.com
    description: STAGING environment
  - url: https://b.example.com
    description: Local development server
  - url: https://c.example.com
    description: Sandbox for partners
`), nil)
	})

	t.Run("matches names inside words like the original", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://a.example.com
    description: Contested device server
`), nil)
	})

	t.Run("honours the pattern option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
servers:
  - url: https://a.example.com
    description: The main server
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"pattern": "main"})
		ruletest.RequireClean(t, rule, document, ruletest.Options{"pattern": ""})
	})

	t.Run("ignores documents without servers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
`), nil)
	})
}
