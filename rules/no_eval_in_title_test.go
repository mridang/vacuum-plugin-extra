package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoEvalInTitle(t *testing.T) {
	t.Parallel()

	rule := rules.NoEvalInTitle()

	t.Run("flags titles and descriptions containing eval", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, `
openapi: 3.0.3
info:
  title: Pets EVAL(1)
  version: "1.0"
paths:
  /pets:
    get:
      description: run eval(x) now
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      title: Pet
      properties:
        description:
          type: string
          description: clean
`, nil)

		require.ElementsMatch(t, []string{
			"$.info.title",
			"$.paths['/pets'].get.description",
		}, ruletest.Paths(results))
	})

	t.Run("accepts text that only mentions evaluate", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: Evaluate (carefully) the query
      responses:
        "200":
          description: ok
`), nil)
	})
}
