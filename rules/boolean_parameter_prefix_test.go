package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestBooleanParameterPrefix(t *testing.T) {
	t.Parallel()

	rule := rules.BooleanParameterPrefix()

	t.Run("flags a boolean parameter without a prefix", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: active
          in: query
          schema:
            type: boolean
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.parameters[0].name", results[0].Path)
		require.Contains(t, results[0].Message, "active")
	})

	t.Run("accepts the default prefixes", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: isActive
          in: query
          schema:
            type: boolean
        - name: has_tail
          in: query
          schema:
            type: boolean
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("rejects words that merely start with a prefix", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  parameters:
    Island:
      name: island
      in: query
      schema:
        type: boolean
`), nil, "$.components.parameters.Island.name")
	})

	t.Run("ignores non boolean parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  parameters:
    Limit:
      name: limit
      in: query
      schema:
        type: integer
`), nil)
	})

	t.Run("follows schema references", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: enabled
        in: query
        schema:
          $ref: "#/components/schemas/Flag"
components:
  schemas:
    Flag:
      type: boolean
`), nil, "$.paths['/pets'].parameters[0].name")
	})

	t.Run("honours custom prefixes", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
components:
  parameters:
    Active:
      name: canEdit
      in: query
      schema:
        type: boolean
`)

		ruletest.RequireClean(t, rule, doc, ruletest.Options{"prefixes": []any{"can"}})
		ruletest.RequirePaths(t, rule, doc, nil, "$.components.parameters.Active.name")
	})
}
