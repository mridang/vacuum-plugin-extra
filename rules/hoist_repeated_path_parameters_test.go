package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestHoistRepeatedPathParameters(t *testing.T) {
	t.Parallel()

	rule := rules.HoistRepeatedPathParameters()

	t.Run("flags an identical path parameter in two operations", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
    delete:
      parameters:
        - schema: {type: string}
          required: true
          in: path
          name: id
      responses:
        "204":
          description: gone
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets/{id}'].get.parameters[0]",
			"$.paths['/pets/{id}'].delete.parameters[0]",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "'id'")
	})

	t.Run("accepts a parameter declared on the path item", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    parameters:
      - {name: id, in: path, required: true, schema: {type: string}}
    get:
      responses:
        "200":
          description: ok
    delete:
      responses:
        "204":
          description: gone
`), nil)
	})

	t.Run("ignores parameters that differ", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, description: one, schema: {type: string}}
      responses:
        "200":
          description: ok
    delete:
      parameters:
        - {name: id, in: path, required: true, description: two, schema: {type: string}}
      responses:
        "204":
          description: gone
`), nil)
	})

	t.Run("ignores repeated query parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: q, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
    post:
      parameters:
        - {name: q, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("resolves referenced parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets/{id}:
    get:
      parameters:
        - $ref: "#/components/parameters/Id"
    put:
      parameters:
        - $ref: "#/components/parameters/Id"
components:
  parameters:
    Id: {name: id, in: path, required: true, schema: {type: string}}
`), nil, "$.paths['/pets/{id}'].get.parameters[0]", "$.paths['/pets/{id}'].put.parameters[0]")
	})
}
