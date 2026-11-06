package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAllowReservedQueryOnly(t *testing.T) {
	t.Parallel()

	rule := rules.AllowReservedQueryOnly()

	t.Run("flags allowReserved on path and header parameters", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          allowReserved: true
          schema:
            type: string
        - name: X-Trace
          in: header
          allowReserved: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets/{id}'].get.parameters[0].allowReserved",
			"$.paths['/pets/{id}'].get.parameters[1].allowReserved",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "query")
	})

	t.Run("accepts allowReserved on query parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          allowReserved: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags response headers and referenced parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Id"
      responses:
        "200":
          description: ok
          headers:
            X-Rate:
              allowReserved: true
              schema:
                type: string
components:
  parameters:
    Id:
      name: id
      in: cookie
      allowReserved: true
      schema:
        type: string
`), nil,
			"$.paths['/pets'].get.responses['200'].headers['X-Rate'].allowReserved",
			"$.components.parameters.Id.allowReserved",
		)
	})

	t.Run("ignores allowReserved false", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          allowReserved: false
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})
}
