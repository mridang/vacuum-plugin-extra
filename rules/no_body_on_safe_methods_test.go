package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoBodyOnSafeMethods(t *testing.T) {
	t.Parallel()

	rule := rules.NoBodyOnSafeMethods()

	t.Run("flags a request body on get and delete", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "200":
          description: ok
    delete:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "204":
          description: gone
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          description: created
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.requestBody",
			"$.paths['/pets'].delete.requestBody",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "request body")
	})

	t.Run("accepts operations without a request body", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the httpMethods option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
paths:
  /pets:
    get:
      requestBody:
        content:
          application/json: {}
    post:
      requestBody:
        content:
          application/json: {}
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"httpMethods": []string{"POST"}},
			"$.paths['/pets'].post.requestBody")
		ruletest.RequireClean(t, rule, document, ruletest.Options{"httpMethods": []string{"put"}})
	})
}
