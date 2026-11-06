package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const noDefaultForRequiredParameterDoc = `
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          required: true
          schema: {type: integer, default: 10}
        - name: flag
          in: query
          required: true
          schema: {type: boolean, default: false}
        - name: ref
          in: query
          required: true
          schema: {$ref: "#/components/schemas/Limit"}
        - name: optional
          in: query
          schema: {type: integer, default: 1}
      responses:
        "200":
          description: ok
components:
  schemas:
    Limit: {type: integer, default: 5}
  parameters:
    Shared:
      name: shared
      in: query
      required: true
      schema: {type: string, default: x}
`

func TestNoDefaultForRequiredParameter(t *testing.T) {
	t.Parallel()

	rule := rules.NoDefaultForRequiredParameter()

	t.Run("flags any default on required parameters", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(noDefaultForRequiredParameterDoc), nil,
			"$.paths['/pets'].get.parameters[0].schema.default",
			"$.paths['/pets'].get.parameters[1].schema.default",
			"$.paths['/pets'].get.parameters[2].schema",
			"$.components.parameters.Shared.schema.default")
	})

	t.Run("onlyTruthy tolerates falsy defaults", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(noDefaultForRequiredParameterDoc),
			ruletest.Options{"onlyTruthy": true},
			"$.paths['/pets'].get.parameters[0].schema.default",
			"$.paths['/pets'].get.parameters[2].schema",
			"$.components.parameters.Shared.schema.default")
	})

	t.Run("accepts required parameters without a default", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema: {type: string}
      responses:
        "200":
          description: ok
`), nil)
	})
}
