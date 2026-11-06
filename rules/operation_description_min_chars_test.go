package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationDescriptionMinChars(t *testing.T) {
	t.Parallel()

	rule := rules.OperationDescriptionMinChars()

	t.Run("flags operations whose texts are short", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List pets
      description: Lists them
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.description", results[0].Path)
	})

	t.Run("accepts a long description even with a short summary", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List
      description: Returns every pet that the caller is allowed to see.
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores operations without either field", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the minLength option", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, doc, ruletest.Options{"minLength": 5})
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"minLength": 50}, "$.paths['/pets'].get.summary")
	})
}
