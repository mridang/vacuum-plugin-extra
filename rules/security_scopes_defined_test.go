package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const securityScopesDefinedSchemes = `
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/auth
          scopes:
            read: Read access
            write: Write access
    key:
      type: apiKey
      name: k
      in: header
`

func TestSecurityScopesDefined(t *testing.T) {
	t.Parallel()

	rule := rules.SecurityScopesDefined()

	t.Run("flags undefined schemes and scopes", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
security:
  - oauth: [read, admin]
paths:
  /pets:
    get:
      security:
        - missing: []
      responses:
        "200":
          description: ok
`+securityScopesDefinedSchemes), nil)

		require.ElementsMatch(t, []string{
			"$.security[0].oauth[1]",
			"$.paths['/pets'].get.security[0].missing",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "admin")
	})

	t.Run("accepts declared schemes and scopes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
security:
  - oauth: [read]
  - key: []
paths:
  /pets:
    get:
      security:
        - oauth: [write]
      responses:
        "200":
          description: ok
`+securityScopesDefinedSchemes), nil)
	})

	t.Run("reportUnused flags declared scopes that are never used", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
security:
  - oauth: [read]
paths: {}
` + securityScopesDefinedSchemes)

		ruletest.RequireClean(t, rule, document, nil)
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"reportUnused": true},
			"$.components.securitySchemes.oauth.flows.implicit.scopes.write")
	})

	t.Run("an empty requirement is valid", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
security:
  - {}
paths: {}
`), nil)
	})
}
