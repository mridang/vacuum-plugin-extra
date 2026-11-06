package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDescriptionUsageHintWording(t *testing.T) {
	t.Parallel()

	rule := rules.DescriptionUsageHintWording()

	t.Run("flags a long description without a hint word", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: The collection of every pet in the store.
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.description", results[0].Path)
	})

	t.Run("accepts a description with a hint word in any case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: RETURNS the collection of every pet in the store.
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("skips short descriptions and ignores the summary", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: The collection of every pet in the store.
      description: Pet list.
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours custom hint words and minimum length", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: Pet list.
      responses:
        "200":
          description: ok
    post:
      description: Pet list that fetches stuff.
      responses:
        "200":
          description: ok
`), ruletest.Options{"hintWords": []any{"fetches"}, "minLength": 5}, "$.paths['/pets'].get.description")
	})

	t.Run("does nothing without hint words", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: The collection of every pet in the store.
      responses:
        "200":
          description: ok
`), ruletest.Options{"hintWords": []any{}})
	})
}
