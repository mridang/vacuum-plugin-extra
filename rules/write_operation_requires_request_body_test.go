package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestWriteOperationRequiresRequestBody(t *testing.T) {
	t.Parallel()

	rule := rules.WriteOperationRequiresRequestBody()
	doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
    post:
      responses:
        "201":
          description: created
    put:
      requestBody: {}
      responses:
        "200":
          description: ok
    patch:
      responses:
        "200":
          description: ok
    delete:
      responses:
        "204":
          description: ok
`)

	t.Run("flags write operations without a requestBody", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].post", "$.paths['/pets'].patch")
	})

	t.Run("honours the methods option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"methods": []any{"delete"}}, "$.paths['/pets'].delete")
	})

	t.Run("accepts operations with bodies", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema: {type: object}
      responses:
        "201":
          description: created
`), nil)
	})
}
