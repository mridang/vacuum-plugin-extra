package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestParameterSchemaTypeRequired(t *testing.T) {
	t.Parallel()

	rule := rules.ParameterSchemaTypeRequired()

	t.Run("flags missing schema and missing type", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: trace
        in: header
        content:
          text/plain:
            schema:
              type: string
    get:
      parameters:
        - name: limit
          in: query
          schema:
            minimum: 1
        - name: offset
          in: query
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0]",
			"$.paths['/pets'].get.parameters[0].schema",
			"$.paths['/pets'].get.parameters[1]",
		}, ruletest.Paths(results))
	})

	t.Run("accepts typed parameters including referenced ones", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
        - name: kind
          in: query
          schema:
            type: [string, "null"]
        - $ref: "#/components/parameters/Offset"
      responses:
        "200":
          description: ok
components:
  parameters:
    Offset:
      name: offset
      in: query
      schema:
        $ref: "#/components/schemas/Count"
  schemas:
    Count:
      type: integer
`), nil)
	})

	t.Run("flags a referenced parameter on the reference", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Offset"
      responses:
        "200":
          description: ok
components:
  parameters:
    Offset:
      name: offset
      in: query
      schema:
        description: no type
`), nil, "$.paths['/pets'].get.parameters[0]")
	})
}
