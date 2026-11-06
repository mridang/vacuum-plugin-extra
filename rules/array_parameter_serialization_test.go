package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestArrayParameterSerialization(t *testing.T) {
	t.Parallel()

	rule := rules.ArrayParameterSerialization()

	t.Run("flags a missing style and a missing explode", func(t *testing.T) {
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

		require.Len(t, results, 2)
		require.Equal(t, "$.paths['/pets'].get.parameters[0]", results[0].Path)
		require.Contains(t, ruletest.Messages(results)[0], "style")
		require.Contains(t, ruletest.Messages(results)[1], "explode")
	})

	t.Run("flags a path level parameter and a component parameter", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: ids
        in: query
        style: form
        schema:
          type: array
    get:
      responses:
        "200":
          description: ok
components:
  parameters:
    Tags:
      name: tags
      in: query
      explode: true
      schema:
        type: array
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0]",
			"$.components.parameters.Tags",
		}, ruletest.Paths(results))
	})

	t.Run("accepts a fully serialized array and non array parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: query
          style: form
          explode: false
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

	t.Run("limits the check to configured locations", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: tags
          in: header
          schema:
            type: array
      responses:
        "200":
          description: ok
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"in": []string{"query"}})
		require.Len(t, ruletest.Run(t, rule, document, nil), 2)
	})

	t.Run("resolves referenced parameters and schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Tags"
      responses:
        "200":
          description: ok
components:
  schemas:
    List:
      type: array
  parameters:
    Tags:
      name: tags
      in: query
      style: form
      schema:
        $ref: "#/components/schemas/List"
`), nil, "$.paths['/pets'].get.parameters[0]")
	})
}
