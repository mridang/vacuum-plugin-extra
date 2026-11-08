package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const pathParameterCaseDocument = `
paths:
  /pets/{petId}:
    get:
      responses:
        "200":
          description: ok
  /owners/{owner_id}/pets/{pet_id}:
    get:
      responses:
        "200":
          description: ok
  /tags:
    get:
      responses:
        "200":
          description: ok
`

func TestPathParameterCase(t *testing.T) {
	t.Parallel()

	rule := rules.PathParameterCase()

	t.Run("flags a camelCase variable by default", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(pathParameterCaseDocument), nil)

		require.Equal(t, []string{"$.paths['/pets/{petId}']"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "petId")
	})

	t.Run("honours the casing option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(pathParameterCaseDocument),
			ruletest.Options{"casing": "camelCase"},
			"$.paths['/owners/{owner_id}/pets/{pet_id}']",
			"$.paths['/owners/{owner_id}/pets/{pet_id}']",
		)
	})

	t.Run("accepts snake case and paths without variables", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /owners/{owner_id}:
    get:
      responses:
        "200":
          description: ok
  /tags:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}
