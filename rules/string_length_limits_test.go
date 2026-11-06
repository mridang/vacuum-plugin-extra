package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestStringLengthLimits(t *testing.T) {
	t.Parallel()

	rule := rules.StringLengthLimits()

	t.Run("flags a non positive minLength and an oversized maxLength", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Zero:
      type: string
      minLength: 0
    Huge:
      type: string
      maxLength: 1073741824
    Both:
      type: string
      minLength: 5000000
      maxLength: 5000001
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Zero.minLength",
			"$.components.schemas.Huge.maxLength",
			"$.components.schemas.Both.minLength",
			"$.components.schemas.Both.maxLength",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results), "minLength must be at least 1")
		require.Contains(t, ruletest.Messages(results), "maxLength must not exceed 4194304")
	})

	t.Run("accepts sensible lengths", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Name:
      type: string
      minLength: 1
      maxLength: 4194304
`), nil)
	})

	t.Run("flags non integer and negative values", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Frac:
      type: string
      minLength: 1.5
    Negative:
      type: string
      maxLength: -1
      minLength: -2
`), ruletest.Options{"minLengthFloor": 0},
			"$.components.schemas.Frac.minLength")
	})

	t.Run("honours the options on inline schemas", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          schema:
            type: string
            minLength: 0
            maxLength: 100
      responses:
        "200":
          description: ok
`)
		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/pets'].get.parameters[0].schema.minLength")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"minLengthFloor": 0})
		ruletest.RequirePaths(t, rule, doc, ruletest.Options{"minLengthFloor": 0, "lengthCeiling": 50},
			"$.paths['/pets'].get.parameters[0].schema.maxLength")
	})
}
