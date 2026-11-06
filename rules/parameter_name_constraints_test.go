package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestParameterNameConstraints(t *testing.T) {
	t.Parallel()

	rule := rules.ParameterNameConstraints()

	t.Run("flags empty and overlong names", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: `+strings.Repeat("x", 51)+`
        in: query
        schema:
          type: string
    get:
      parameters:
        - name: ""
          in: query
          schema:
            type: string
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].parameters[0].name",
			"$.paths['/pets'].get.parameters[0].name",
		}, ruletest.Paths(results))
		require.Contains(t, strings.Join(ruletest.Messages(results), "|"), "at least 1")
	})

	t.Run("accepts ordinary names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("applies the pattern option to component parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  parameters:
    Bad:
      name: page size
      in: query
      schema:
        type: integer
    Good:
      name: page_size
      in: query
      schema:
        type: integer
`), ruletest.Options{"pattern": `^[A-Za-z0-9_]+$`}, "$.components.parameters.Bad.name")
	})

	t.Run("honours the length options", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxLength": 3}, "$.paths['/pets'].get.parameters[0].name")
	})
}
