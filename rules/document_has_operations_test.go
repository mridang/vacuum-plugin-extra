package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDocumentHasOperations(t *testing.T) {
	t.Parallel()

	rule := rules.DocumentHasOperations()

	t.Run("flags empty paths", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters: []
`), nil, "$.paths")
	})

	t.Run("flags a document without paths", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components: {}
`), nil, "$")
	})

	t.Run("accepts a document with an operation", func(t *testing.T) {
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

	t.Run("accepts operations declared only as webhooks", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
webhooks:
  newPet:
    post:
      responses:
        "200":
          description: ok
`), nil)
	})
}
