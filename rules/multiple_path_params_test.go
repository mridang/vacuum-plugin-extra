package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestMultiplePathParams(t *testing.T) {
	t.Parallel()

	rule := rules.MultiplePathParams()
	doc := ruletest.Doc30(`
paths:
  /repos/{repo}:
    get:
      responses:
        "200":
          description: ok
  /orgs/{org}/repos/{repo}:
    get:
      responses:
        "200":
          description: ok
  /a/{a}/b/{b}/c/{c}:
    get:
      responses:
        "200":
          description: ok
`)

	t.Run("flags paths with more than one parameter", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, doc, nil,
			"$.paths['/orgs/{org}/repos/{repo}']", "$.paths['/a/{a}/b/{b}/c/{c}']")
	})

	t.Run("honours the max option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"max": 2}, "$.paths['/a/{a}/b/{b}/c/{c}']")
	})

	t.Run("accepts paths with at most one parameter", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}
