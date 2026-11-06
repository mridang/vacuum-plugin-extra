package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestQueryArrayStyleForm(t *testing.T) {
	t.Parallel()

	rule := rules.QueryArrayStyleForm()

	t.Run("flags an array query parameter without style", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          schema:
            type: array
            items:
              type: string
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.parameters[0]", results[0].Path)
		require.Contains(t, results[0].Message, "tags")
	})

	t.Run("flags a different style", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          style: pipeDelimited
          schema:
            type: array
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].get.parameters[0]")
	})

	t.Run("accepts form and ignores other parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          style: form
          schema:
            type: array
        - name: ids
          in: header
          schema:
            type: array
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("follows referenced schemas and path level parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: ids
        in: query
        schema:
          $ref: "#/components/schemas/Ids"
    get:
      responses:
        "200":
          description: ok
components:
  schemas:
    Ids:
      type: array
      items:
        type: string
`), nil, "$.paths['/pets'].parameters[0]")
	})
}
