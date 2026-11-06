package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPatternAnchored(t *testing.T) {
	t.Parallel()

	rule := rules.PatternAnchored()

	t.Run("flags unanchored patterns with the matching message", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Both:
      type: string
      pattern: "[A-Z]{3}"
    NoStart:
      type: string
      pattern: "[A-Z]{3}$"
    NoEnd:
      type: string
      pattern: "^[A-Z]{3}"
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Both.pattern",
			"$.components.schemas.NoStart.pattern",
			"$.components.schemas.NoEnd.pattern",
		}, ruletest.Paths(results))
		require.ElementsMatch(t, []string{
			"pattern must start with ^ and end with $",
			"pattern must start with ^",
			"pattern must end with $",
		}, ruletest.Messages(results))
	})

	t.Run("accepts anchored patterns", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Code:
      type: string
      pattern: "^[A-Z]{3}$"
`), nil)
	})

	t.Run("treats an escaped dollar as unanchored", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Price:
      type: string
      pattern: '^\$'
    Slash:
      type: string
      pattern: '^a\\$'
`), nil, "$.components.schemas.Price.pattern")
	})

	t.Run("checks composition members and skips non string types", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Name:
      type: string
      allOf:
        - pattern: "^a"
      oneOf:
        - pattern: "^[a-z]+$"
    Odd:
      type: integer
      pattern: "x"
`), nil, "$.components.schemas.Name.allOf[0].pattern")
	})

	t.Run("honours requireStart and requireEnd", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
components:
  schemas:
    Code:
      type: string
      pattern: "[A-Z]{3}$"
`)
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"requireStart": false})
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"requireEnd": false}, "$.components.schemas.Code.pattern")
	})
}
