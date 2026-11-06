package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoLinkHeader(t *testing.T) {
	t.Parallel()

	rule := rules.NoLinkHeader()

	t.Run("flags a Link response header and a Link header parameter", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: link
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
          headers:
            Link:
              schema:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.parameters[0]",
			"$.paths['/pets'].get.responses['200'].headers.Link",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "Link")
	})

	t.Run("flags path level and component definitions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    parameters:
      - name: LINK
        in: header
        schema:
          type: string
    get:
      responses:
        "200":
          description: ok
components:
  responses:
    Page:
      description: page
      headers:
        link:
          schema:
            type: string
  headers:
    Link:
      schema:
        type: string
`), nil,
			"$.paths['/pets'].parameters[0]",
			"$.components.responses.Page.headers.link",
			"$.components.headers.Link",
		)
	})

	t.Run("ignores a query parameter named link and other headers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: link
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
          headers:
            Location:
              schema:
                type: string
`), nil)
	})
}
