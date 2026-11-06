package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const securityScopesDeclaredOAuth = `
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes:
            pets.read: read pets
`

const securityScopesDeclaredOAuthSchemes = `
    oauth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes:
            pets.read: read pets
`

func TestSecurityScopesDeclared(t *testing.T) {
	t.Parallel()

	rule := rules.SecurityScopesDeclared()

	t.Run("flags undeclared scopes in operations and at the document level", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
security:
  - oauth: [pets.admin]
paths:
  /pets:
    get:
      security:
        - oauth: [pets.read, pets.write]
      responses:
        "200":
          description: ok
`+securityScopesDeclaredOAuth), nil)

		require.ElementsMatch(t, []string{
			"$.security[0].oauth[0]",
			"$.paths['/pets'].get.security[0].oauth[1]",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "not declared")
	})

	t.Run("flags a missing OAuth2 or bearer scheme", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.securitySchemes", results[0].Path)
	})

	t.Run("flags other scheme types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
    bearer:
      type: http
      scheme: Bearer
    basic:
      type: http
      scheme: basic
`), nil, "$.components.securitySchemes.key", "$.components.securitySchemes.basic")
	})

	t.Run("names the type of a scheme that is not allowed, or says it has none", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  securitySchemes:
    cookie:
      type: apiKey
      name: sid
      in: cookie
    typeless:
      description: no type
`+securityScopesDeclaredOAuthSchemes), nil)

		require.ElementsMatch(t, []string{
			"security scheme 'cookie' of type apiKey is not allowed, use oauth2 or bearer",
			"security scheme 'typeless' of type (none) is not allowed, use oauth2 or bearer",
		}, ruletest.Messages(results))
	})

	t.Run("accepts declared scopes and non OAuth2 requirement scopes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      security:
        - oauth: [pets.read]
        - bearer: [anything]
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes:
            pets.read: read pets
    bearer:
      type: http
      scheme: bearer
`), nil)
	})

	t.Run("honours the allowedTypes option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
`), ruletest.Options{"allowedTypes": []string{"apiKey"}})
	})
}
