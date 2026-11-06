package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestCORSOriginHeader(t *testing.T) {
	t.Parallel()

	rule := rules.CORSOriginHeader()

	t.Run("flags responses without headers or without the header", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
        "404":
          description: missing
          headers:
            X-Request-Id:
              schema:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200']",
			"$.paths['/pets'].get.responses['404']",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "Access-Control-Allow-Origin")
	})

	t.Run("accepts the header in any letter case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            access-control-allow-origin:
              schema:
                type: string
`), nil)
	})

	t.Run("resolves referenced responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          $ref: "#/components/responses/Ok"
components:
  responses:
    Ok:
      description: ok
      headers:
        Access-Control-Allow-Origin:
          schema:
            type: string
`), nil)
	})

	t.Run("honours the header option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            Access-Control-Allow-Origin:
              schema:
                type: string
`), ruletest.Options{"header": "Vary"}, "$.paths['/pets'].get.responses['200']")
	})
}
