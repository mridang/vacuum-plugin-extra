package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPatternLengthSatisfiable(t *testing.T) {
	t.Parallel()

	rule := rules.PatternLengthSatisfiable()

	t.Run("flags a pattern shorter than minLength", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Code:
      type: string
      pattern: "^[A-Z]{3}$"
      minLength: 5
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Code.pattern", results[0].Path)
		require.Contains(t, results[0].Message, "minLength")
	})

	t.Run("flags a pattern longer than maxLength", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Code:
      type: string
      pattern: "^[a-z]{4,8}-[0-9]+$"
      maxLength: 5
`), nil, "$.components.schemas.Code.pattern")
	})

	t.Run("accepts satisfiable combinations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: string
      pattern: "^[A-Z]{3}$"
      minLength: 3
      maxLength: 3
    B:
      type: string
      pattern: "^[a-z]+$"
      minLength: 2
      maxLength: 10
    C:
      type: string
      pattern: "^(ab|abcdef)$"
      minLength: 6
`), nil)
	})

	t.Run("treats unanchored patterns as matching longer strings", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Free:
      type: string
      pattern: "[A-Z]{3}"
      minLength: 10
`), nil)

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Free:
      type: string
      pattern: "[A-Z]{3}"
      maxLength: 2
`), nil, "$.components.schemas.Free.pattern")
	})

	t.Run("considers every alternative", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Alt:
      type: string
      pattern: "^(ab|cd)$"
      minLength: 3
`), nil, "$.components.schemas.Alt.pattern")
	})

	t.Run("skips patterns RE2 cannot parse and schemas without limits", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Look:
      type: string
      pattern: "^(?=a)b$"
      minLength: 9
    Plain:
      type: string
      pattern: "^abc$"
`), nil)
	})

	t.Run("flags contradictory anchors and classes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Anchors:
      type: string
      pattern: "^a$b"
      minLength: 1
    Middle:
      type: string
      pattern: "a^b"
      maxLength: 9
    Digits:
      type: string
      pattern: "^[0-9]{2}[^0-9a-z]{2}$"
      minLength: 5
      maxLength: 8
    Fine:
      type: string
      pattern: "^[0-9]{2}[^0-9a-z]{2}$"
      minLength: 4
`), nil,
			"$.components.schemas.Anchors.pattern",
			"$.components.schemas.Middle.pattern",
			"$.components.schemas.Digits.pattern")
	})

	t.Run("handles repetition counts, case folding and large limits", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Exact:
      type: string
      pattern: "^(?i)ab{2}c?$"
      minLength: 5
    Wide:
      type: string
      pattern: "^.*$"
      minLength: 100
      maxLength: 5000
    Odd:
      type: string
      pattern: "^(ab){3}$"
      minLength: 4
      maxLength: 5
`), nil,
			"$.components.schemas.Exact.pattern",
			"$.components.schemas.Odd.pattern")
	})
}
