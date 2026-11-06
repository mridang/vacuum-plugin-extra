package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const unreachable405AllMethods = `
paths:
  /pets:
    get:
      responses:
        "405":
          description: not allowed
    put:
      responses:
        "405":
          description: not allowed
    post:
      responses:
        "200":
          description: ok
    delete:
      responses:
        "200":
          description: ok
    options:
      responses:
        "200":
          description: ok
    head:
      responses:
        "200":
          description: ok
    patch:
      responses:
        "200":
          description: ok
    trace:
      responses:
        "200":
          description: ok
`

func TestUnreachable405(t *testing.T) {
	t.Parallel()

	rule := rules.Unreachable405()

	t.Run("flags 405 when all methods are declared", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(unreachable405AllMethods), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['405']",
			"$.paths['/pets'].put.responses['405']",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "/pets")
	})

	t.Run("accepts 405 when a method is missing", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "405":
          description: not allowed
    post:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the methods option", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "405":
          description: not allowed
    post:
      responses:
        "200":
          description: ok
`)
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"methods": []any{"GET", "post"}},
			"$.paths['/pets'].get.responses['405']")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"methods": []any{}})
	})

	t.Run("ignores complete paths without a 405", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), ruletest.Options{"methods": []any{"get"}})
	})
}
