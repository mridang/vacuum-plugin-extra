package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestActionVerbsInPath(t *testing.T) {
	t.Parallel()

	rule := rules.ActionVerbsInPath()

	t.Run("flags verb segments per operation", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/create:
    post:
      responses:
        "201":
          description: ok
  /Delete/{id}:
    delete:
      responses:
        "204":
          description: ok
`), nil, "$.paths['/pets/create'].post", "$.paths['/Delete/{id}'].delete")
	})

	t.Run("accepts noun paths and plural look-alikes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /settings/{id}:
    get:
      responses:
        "200":
          description: ok
  /lists:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours a custom verb list", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets/create:
    post:
      responses:
        "201":
          description: ok
  /pets/cancel:
    post:
      responses:
        "200":
          description: ok
`)
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"verbs": []any{"cancel"}}, "$.paths['/pets/cancel'].post")
	})
}
