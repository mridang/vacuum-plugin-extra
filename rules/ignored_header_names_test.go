package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestIgnoredHeaderNames(t *testing.T) {
	t.Parallel()

	rule := rules.IgnoredHeaderNames()

	t.Run("flags ignored request and response headers", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: authorization
        in: header
        schema:
          type: string
    post:
      parameters:
        - name: Content-Type
          in: header
          schema:
            type: string
        - name: X-Trace
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
          headers:
            content-type:
              schema:
                type: string
            X-Rate:
              schema:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0]",
			"$.paths['/pets'].post.parameters[0]",
			"$.paths['/pets'].post.responses['200'].headers['content-type']",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "ignored")
	})

	t.Run("accepts ordinary headers and same-named query parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: X-Trace
          in: header
          schema:
            type: string
        - name: Accept
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the names option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: Accept
          in: header
          schema:
            type: string
        - name: Content-Type
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), ruletest.Options{"names": []any{"Content-Type"}}, "$.paths['/pets'].get.parameters[1]")
	})
}
