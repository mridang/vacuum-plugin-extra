package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationDescriptionRestatesPath(t *testing.T) {
	t.Parallel()

	rule := rules.OperationDescriptionRestatesPath()

	t.Run("flags descriptions and summaries that restate the request line", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: "  GET /Pets "
      description: /pets
      responses:
        "200":
          description: ok
    post:
      description: post/pets
      responses:
        "201":
          description: created
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.summary",
			"$.paths['/pets'].get.description",
			"$.paths['/pets'].post.description",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "restates")
	})

	t.Run("accepts meaningful text", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      summary: List all pets
      description: Gets /pets in pages of ten
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("does not mix up methods", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets/{id}:
    delete:
      summary: get /pets/{id}
      description: DELETE /pets/{id}
      responses:
        "204":
          description: gone
`), nil, "$.paths['/pets/{id}'].delete.description")
	})

	t.Run("ignores operations without text", func(t *testing.T) {
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
}
