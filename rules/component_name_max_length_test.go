package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestComponentNameMaxLength(t *testing.T) {
	t.Parallel()

	rule := rules.ComponentNameMaxLength()

	t.Run("flags names over the default limit in several sections", func(t *testing.T) {
		t.Parallel()

		long := "AVeryLongComponentNameThatDefinitelyExceedsFiftyCharactersInTotal"
		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    `+long+`:
      type: object
    Pet:
      type: object
  responses:
    `+long+`:
      description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas." + long,
			"$.components.responses." + long,
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "maximum is 50")
	})

	t.Run("accepts names within the limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
`), nil)
	})

	t.Run("honours the maxLength option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  parameters:
    PageSize:
      name: size
      in: query
      schema:
        type: integer
`), ruletest.Options{"maxLength": 4}, "$.components.parameters.PageSize")
	})

	t.Run("treats the limit as inclusive", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  headers:
    Rate:
      schema:
        type: integer
`), ruletest.Options{"maxLength": 4})
	})

	t.Run("does nothing when there are no components", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
`), nil)
	})
}
