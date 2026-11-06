package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestResponseContainsHeader(t *testing.T) {
	t.Parallel()

	rule := rules.ResponseContainsHeader()

	const document = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            x-request-id:
              schema:
                type: string
        "404":
          description: missing
        default:
          description: other
`

	t.Run("flags responses missing a configured header", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(document), ruletest.Options{
			"headers": map[string]any{
				"200":     []any{"X-Request-Id", "X-Rate-Limit"},
				"4XX":     []any{"Retry-After"},
				"default": "X-Trace",
			},
		})

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200']",
			"$.paths['/pets'].get.responses['404']",
			"$.paths['/pets'].get.responses.default",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results), "response 200 must declare the X-Rate-Limit header")
	})

	t.Run("accepts responses that declare the headers case-insensitively", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(document), ruletest.Options{
			"headers": map[string][]string{"200": {"X-Request-ID"}},
		})
	})

	t.Run("resolves referenced responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
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
`), ruletest.Options{"headers": map[string]any{"200": []any{"X-Request-Id"}}}, "$.paths['/pets'].get.responses['200']")
	})

	t.Run("does nothing without configuration", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(document), nil)
	})
}
