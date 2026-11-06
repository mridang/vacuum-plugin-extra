package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOAuth2RefreshURL(t *testing.T) {
	t.Parallel()

	rule := rules.OAuth2RefreshURL()

	t.Run("flags flows without a refreshUrl except clientCredentials", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          scopes: {}
        clientCredentials:
          tokenUrl: https://auth.example.com/token
          scopes: {}
        implicit:
          authorizationUrl: https://auth.example.com/authorize
          refreshUrl: ""
          scopes: {}
        password:
          tokenUrl: https://auth.example.com/token
          refreshUrl: https://auth.example.com/refresh
          scopes: {}
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.securitySchemes.oauth.flows.authorizationCode",
			"$.components.securitySchemes.oauth.flows.implicit",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "refreshUrl")
	})

	t.Run("accepts flows with a refreshUrl", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/authorize
          tokenUrl: https://auth.example.com/token
          refreshUrl: https://auth.example.com/refresh
          scopes: {}
`), nil)
	})

	t.Run("ignores other scheme types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  securitySchemes:
    key:
      type: apiKey
      name: X-Key
      in: header
    bearer:
      type: http
      scheme: bearer
`), nil)
	})

	t.Run("treats an empty refreshUrl as missing", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        password:
          tokenUrl: https://auth.example.com/token
          refreshUrl: ""
          scopes: {}
`), nil, "$.components.securitySchemes.oauth.flows.password")
	})
}
