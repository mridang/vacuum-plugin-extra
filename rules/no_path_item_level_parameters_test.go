package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoPathItemLevelParameters(t *testing.T) {
	t.Parallel()

	rule := rules.NoPathItemLevelParameters()

	t.Run("flags path item parameters", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets/{id}'].parameters", results[0].Path)
	})

	t.Run("accepts operation level parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags each offending path item", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /a:
    parameters: []
  /b:
    parameters: []
  /c: {}
`), nil, "$.paths['/a'].parameters", "$.paths['/b'].parameters")
	})
}
