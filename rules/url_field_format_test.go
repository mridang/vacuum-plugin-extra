package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestURLFieldFormat(t *testing.T) {
	t.Parallel()

	rule := rules.URLFieldFormat()

	t.Run("flags malformed and relative urls", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, `
openapi: 3.0.3
info:
  title: Test
  version: "1.0"
  termsOfService: not a url
  contact:
    url: example.com/support
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
externalDocs:
  url: /docs
servers:
  - url: "https://bad host.example.com"
tags:
  - name: pets
    externalDocs:
      url: docs/pets
paths:
  /pets:
    get:
      externalDocs:
        url: ":bad"
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    oidc:
      type: openIdConnect
      openIdConnectUrl: relative/path
    oauth:
      type: oauth2
      flows:
        implicit:
          authorizationUrl: "https://exa mple.com/auth"
          scopes: {}
  schemas:
    Pet:
      type: object
      xml:
        namespace: pets
`, nil)

		require.ElementsMatch(t, []string{
			"$.info.termsOfService",
			"$.info.contact.url",
			"$.externalDocs.url",
			"$.servers[0].url",
			"$.tags[0].externalDocs.url",
			"$.paths['/pets'].get.externalDocs.url",
			"$.components.securitySchemes.oidc.openIdConnectUrl",
			"$.components.securitySchemes.oauth.flows.implicit.authorizationUrl",
			"$.components.schemas.Pet.xml.namespace",
		}, ruletest.Paths(results))
	})

	t.Run("accepts valid urls", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, `
openapi: 3.1.0
info:
  title: Test
  version: "1.0"
  termsOfService: https://example.com/terms
  contact:
    url: https://example.com/support
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
servers:
  - url: https://{region}.example.com/v1
    variables:
      region:
        default: eu
  - url: /relative/base
paths: {}
components:
  securitySchemes:
    oauth:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/auth
          tokenUrl: /token
          scopes: {}
`, nil)
	})
}
