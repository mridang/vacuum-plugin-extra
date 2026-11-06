package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperation2xxRequired(t *testing.T) {
	t.Parallel()

	rule := rules.Operation2xxRequired()

	t.Run("flags operations without a success response", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /a:
    get:
      responses:
        "301":
          description: moved
        default:
          description: error
  /b:
    get:
      summary: no responses
`), nil, "$.paths['/a'].get.responses", "$.paths['/b'].get")
	})

	t.Run("accepts 2xx codes and ranges", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /a:
    get:
      responses:
        "204":
          description: ok
  /b:
    get:
      responses:
        2XX:
          description: ok
`), nil)
	})
}
