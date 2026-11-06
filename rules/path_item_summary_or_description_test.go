package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPathItemSummaryOrDescription(t *testing.T) {
	t.Parallel()

	rule := rules.PathItemSummaryOrDescription()

	t.Run("flags path items with neither", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
  /owners:
    summary: Owners
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets']", results[0].Path)
		require.Contains(t, results[0].Message, "/pets")
	})

	t.Run("accepts a summary or a description", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    summary: Pets
    get:
      responses:
        "200":
          description: ok
  /owners:
    description: The owners of pets.
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("treats blank text as missing and skips extensions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  x-internal: {}
  /pets:
    summary: "  "
    get:
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets']")
	})

	t.Run("follows a path item reference", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    $ref: "#/components/pathItems/Pets"
components:
  pathItems:
    Pets:
      summary: Pets
`), nil)
	})
}
