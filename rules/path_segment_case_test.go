package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPathSegmentCase(t *testing.T) {
	t.Parallel()

	rule := rules.PathSegmentCase()

	t.Run("flags casing, trailing slash, missing slash and empty segments", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /petOwners:
    get:
      responses:
        "200":
          description: ok
  /pets/:
    get:
      responses:
        "200":
          description: ok
  pets2:
    get:
      responses:
        "200":
          description: ok
  /a//b:
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/petOwners']",
			"$.paths['/pets/']",
			"$.paths.pets2",
			"$.paths['/a//b']",
		}, ruletest.Paths(results))
	})

	t.Run("accepts kebab case with parameters and the root", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /:
    get:
      responses:
        "200":
          description: ok
  /pet-owners/{ownerId}/v1:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the casing option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
paths:
  /pet_owners:
    get:
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"casing": "snake_case"})
		ruletest.RequirePaths(t, rule, document, nil, "$.paths['/pet_owners']")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"casing": "PascalCase"}, "$.paths['/pet_owners']")
	})
}
