package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestUniqueTagNames(t *testing.T) {
	t.Parallel()

	rule := rules.UniqueTagNames()

	t.Run("flags every repeated tag", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
tags:
  - name: pets
  - name: owners
  - name: pets
  - name: pets
paths: {}
`), nil)

		require.ElementsMatch(t, []string{"$.tags[2].name", "$.tags[3].name"}, ruletest.Paths(results))
	})

	t.Run("accepts unique tags and documents without tags", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
tags:
  - name: pets
  - name: Pets
  - description: nameless
paths: {}
`), nil)
		ruletest.RequireClean(t, rule, ruletest.Doc30(`paths: {}`), nil)
	})

	t.Run("can compare case insensitively", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
tags:
  - name: pets
  - name: " Pets"
`), ruletest.Options{"caseInsensitive": true}, "$.tags[1].name")
	})
}
