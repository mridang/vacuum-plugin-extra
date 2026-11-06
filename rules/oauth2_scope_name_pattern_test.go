package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const oauth2ScopeNamePatternDocument = `
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/auth
          scopes:
            uid: identity
            pets.read: Read pets
            Admin: Administrator
        clientCredentials:
          tokenUrl: https://example.com/token
          scopes:
            pets.delete: Delete pets
    key:
      type: apiKey
      name: k
      in: header
`

const oauth2ScopeNamePatternEndpointDocument = `
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/auth
          scopes:
            pets.read: Read pets
    key:
      type: apiKey
      name: k
      in: header
`

func TestOAuth2ScopeNamePattern(t *testing.T) {
	t.Parallel()

	rule := rules.OAuth2ScopeNamePattern()

	t.Run("flags scopes that do not match the default pattern", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(oauth2ScopeNamePatternDocument), nil)

		require.ElementsMatch(t, []string{
			"$.components.securitySchemes.oauth.flows.implicit.scopes.Admin",
			"$.components.securitySchemes.oauth.flows.clientCredentials.scopes['pets.delete']",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "does not match regex")
	})

	t.Run("accepts conforming scopes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/auth
          scopes:
            uid: identity
            pets.write: Write pets
`), nil)
	})

	t.Run("honours the pattern and allowed options", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(oauth2ScopeNamePatternDocument),
			ruletest.Options{"pattern": `^[a-z]+\.[a-z]+$`, "allowed": []any{"Admin"}},
			"$.components.securitySchemes.oauth.flows.implicit.scopes.uid",
		)
	})

	t.Run("an invalid pattern disables the rule", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(oauth2ScopeNamePatternDocument),
			ruletest.Options{"pattern": "([unclosed"})
	})

	t.Run("requires a full match of the scope name", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: https://example.com/auth
          scopes:
            uid-extra: identity
            area.pets.read: ok
            a.b.c.read: too deep
`), nil,
			"$.components.securitySchemes.oauth.flows.implicit.scopes['uid-extra']",
			"$.components.securitySchemes.oauth.flows.implicit.scopes['a.b.c.read']",
		)
	})

	t.Run("flags endpoints without any security", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(oauth2ScopeNamePatternEndpointDocument+`
paths:
  /open:
    get:
      responses:
        "200":
          description: ok
  /empty:
    get:
      security: []
      responses:
        "200":
          description: ok
`), nil, "$.paths['/open'].get", "$.paths['/empty'].get.security")
	})

	t.Run("inherits the global security", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(oauth2ScopeNamePatternEndpointDocument+`
security:
  - oauth: [pets.read]
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags oauth2 requirements without or with undefined scopes", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(oauth2ScopeNamePatternEndpointDocument+`
paths:
  /none:
    get:
      security:
        - oauth: []
      responses:
        "200":
          description: ok
  /unknown:
    get:
      security:
        - oauth: [pets.read, pets.nope]
      responses:
        "200":
          description: ok
  /good:
    get:
      security:
        - oauth: [pets.read]
        - key: []
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/none'].get.security", "$.paths['/unknown'].get.security"},
			ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "not secured by OAuth2 scope(s)")
		require.Contains(t, ruletest.Messages(results)[1], "undefined OAuth2 scope(s): oauth:pets.nope")
	})

	t.Run("flags a requirement for an unknown scheme", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(oauth2ScopeNamePatternEndpointDocument+`
paths:
  /pets:
    get:
      security:
        - ghost: []
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/pets'].get.security"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "Security scheme ghost not found")
	})

	t.Run("skips whitelisted paths", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(oauth2ScopeNamePatternEndpointDocument + `
paths:
  /health:
    get:
      responses:
        "200":
          description: ok
`)

		ruletest.RequirePaths(t, rule, document, nil, "$.paths['/health'].get")
		ruletest.RequireClean(t, rule, document, ruletest.Options{"pathWhitelist": []any{"^/health"}})
	})
}
