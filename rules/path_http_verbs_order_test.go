package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPathHTTPVerbsOrder(t *testing.T) {
	t.Parallel()

	rule := rules.PathHTTPVerbsOrder()

	t.Run("flags an operation declared too late", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "200":
          description: ok
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get", results[0].Path)
	})

	t.Run("accepts the default order", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
    head:
      responses:
        "200":
          description: ok
    post:
      responses:
        "200":
          description: ok
    delete:
      responses:
        "200":
          description: ok
  /other:
    delete:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("tracks each path item separately", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /a:
    put:
      responses:
        "200":
          description: ok
  /b:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours a custom order", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    post:
      responses:
        "200":
          description: ok
    get:
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, doc, ruletest.Options{"order": []any{"post", "get"}})
		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].get")
	})
}
