package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPathParamPlacement(t *testing.T) {
	t.Parallel()

	rule := rules.PathParamPlacement()

	t.Run("flags a path that starts with a parameter", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /{tenant}/pets: {}
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/{tenant}/pets']", results[0].Path)
		require.Contains(t, results[0].Message, "start with a path parameter")
	})

	t.Run("flags prefixes and suffixes around a parameter", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/id-{id}: {}
  /files/{name}.json: {}
  /pets/{id}: {}
`), nil, "$.paths['/pets/id-{id}']", "$.paths['/files/{name}.json']")
	})

	t.Run("reports both problems of one path", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
paths:
  /{a}/x-{b}: {}
`), nil)

		require.Len(t, results, 2)
	})

	t.Run("accepts well placed parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /tenants/{tenant}/pets/{petId}: {}
  /pets: {}
  /: {}
`), nil)
	})

	t.Run("flags adjacent parameters in one segment", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{a}{b}: {}
`), nil, "$.paths['/pets/{a}{b}']")
	})
}
