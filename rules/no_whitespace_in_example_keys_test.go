package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoWhitespaceInExampleKeys(t *testing.T) {
	t.Parallel()

	rule := rules.NoWhitespaceInExampleKeys()

	t.Run("flags example keys with whitespace", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                a pet:
                  value: {}
                good-one:
                  value: {}
components:
  responses:
    Other:
      description: other
      content:
        application/json:
          examples:
            "tab	key":
              value: {}
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].examples['a pet']",
			"$.components.responses.Other.content['application/json'].examples['tab\tkey']",
		}, ruletest.Paths(results))
	})

	t.Run("accepts keys without whitespace", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                a_pet:
                  value: {}
`), nil)
	})

	t.Run("ignores request body examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            examples:
              a pet:
                value: {}
      responses:
        "201":
          description: created
`), nil)
	})
}
