package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestMajorVersionInPath(t *testing.T) {
	t.Parallel()

	rule := rules.MajorVersionInPath()

	t.Run("flags unversioned paths without a versioned server", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com
paths:
  /pets: {}
  /v1/owners: {}
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets']", results[0].Path)
		require.Contains(t, results[0].Message, "/v1")
	})

	t.Run("accepts a version in the server url", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com/v2
paths:
  /pets: {}
`), nil)
	})

	t.Run("accepts versioned paths", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /v1/pets: {}
  /v1/owners/{id}: {}
`), nil)
	})

	t.Run("flags paths with a different version", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /v1/pets: {}
  /v2/owners: {}
`), nil, "$.paths['/v2/owners']")
	})

	t.Run("does not require consistency when disabled", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /v1/pets: {}
  /v2/owners: {}
`), ruletest.Options{"consistent": false})
	})

	t.Run("ignores the server when allowServerVersion is off", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
servers:
  - url: https://api.example.com/v1
paths:
  /pets: {}
`), ruletest.Options{"allowServerVersion": false}, "$.paths['/pets']")
	})

	t.Run("does not accept look alike segments", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /v/pets: {}
  /version1/pets: {}
`), nil, "$.paths['/v/pets']", "$.paths['/version1/pets']")
	})
}
