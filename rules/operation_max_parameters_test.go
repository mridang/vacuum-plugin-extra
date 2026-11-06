package rules_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func operationMaxParametersDocument(count int, required bool) string {
	var parameters strings.Builder
	for i := range count {
		fmt.Fprintf(
			&parameters,
			"        - name: p%d\n          in: query\n          required: %t\n          schema:\n            type: string\n",
			i,
			required,
		)
	}

	return ruletest.Doc30("paths:\n  /pets:\n    get:\n      parameters:\n" + parameters.String() +
		"      responses:\n        \"200\":\n          description: ok\n")
}

func TestOperationMaxParameters(t *testing.T) {
	t.Parallel()

	rule := rules.OperationMaxParameters()

	t.Run("flags more than fifteen parameters", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, operationMaxParametersDocument(16, false), nil)

		require.Equal(t, []string{"$.paths['/pets'].get"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "16 parameters")
	})

	t.Run("accepts exactly fifteen parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, operationMaxParametersDocument(15, false), nil)
	})

	t.Run("flags more than seven required parameters", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, operationMaxParametersDocument(8, true), nil)

		require.Equal(t, []string{"$.paths['/pets'].get"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "8 required parameters")
	})

	t.Run("honours the limit options", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, operationMaxParametersDocument(3, false),
			ruletest.Options{"maxParameters": 2}, "$.paths['/pets'].get")
		ruletest.RequireClean(t, rule, operationMaxParametersDocument(30, true),
			ruletest.Options{"maxParameters": 0, "maxRequiredParameters": 0})
	})

	t.Run("counts parameters inherited from the path item", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: a
        in: query
        schema:
          type: string
      - name: b
        in: query
        schema:
          type: string
    get:
      parameters:
        - name: b
          in: query
          schema:
            type: string
        - name: c
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), ruletest.Options{"maxParameters": 2}, "$.paths['/pets'].get")
	})
}
