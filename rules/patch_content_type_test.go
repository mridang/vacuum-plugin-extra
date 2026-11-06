package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const patchContentTypeDoc = `
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/json:
            schema: {type: object}
          application/merge-patch+json; charset=utf-8:
            schema: {type: object}
      responses:
        "200":
          description: ok
    put:
      requestBody:
        content:
          application/json:
            schema: {type: object}
      responses:
        "200":
          description: ok
`

func TestPatchContentType(t *testing.T) {
	t.Parallel()

	rule := rules.PatchContentType()

	t.Run("flags non patch media types on PATCH only", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(patchContentTypeDoc), nil)

		require.Len(t, results, 1)
		require.Contains(t, results[0].Message, "application/json")
		require.Contains(t, results[0].Path, "$.paths['/pets/{id}'].patch.requestBody.content")
	})

	t.Run("accepts json patch", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/json-patch+json:
            schema: {type: array}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the allowed option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(patchContentTypeDoc),
			ruletest.Options{"allowed": []any{"application/json", "application/merge-patch+json"}})
	})
}
