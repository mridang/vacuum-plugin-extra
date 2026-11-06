package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationIDReservedWord(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDReservedWord()

	t.Run("flags python keywords", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /imports:
    get:
      operationId: import
      responses:
        "200":
          description: ok
    post:
      operationId: class
      responses:
        "200":
          description: ok
    put:
      operationId: Import
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/imports'].get.operationId",
			"$.paths['/imports'].post.operationId",
		}, ruletest.Paths(results))
	})

	t.Run("accepts ordinary identifiers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /imports:
    get:
      operationId: listImports
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours a custom word list", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /things:
    get:
      operationId: delete
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, doc, nil)
		ruletest.RequirePaths(
			t,
			rule,
			doc,
			ruletest.Options{"words": []any{"delete"}},
			"$.paths['/things'].get.operationId",
		)
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"words": []any{}})
	})
}
