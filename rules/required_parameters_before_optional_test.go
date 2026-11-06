package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRequiredParametersBeforeOptional(t *testing.T) {
	t.Parallel()

	rule := rules.RequiredParametersBeforeOptional()

	t.Run("flags required parameters after optional ones", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
        - {name: id, in: path, required: true, schema: {type: string}}
        - $ref: "#/components/parameters/Key"
      responses:
        "200":
          description: ok
components:
  parameters:
    Key:
      name: key
      in: header
      required: true
      schema: {type: string}
`), nil, "$.paths['/pets/{id}'].get.parameters[1]", "$.paths['/pets/{id}'].get.parameters[2]")
	})

	t.Run("accepts required parameters first", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {name: limit, in: query, schema: {type: integer}}
        - {name: page, in: query, required: false, schema: {type: integer}}
      responses:
        "200":
          description: ok
`), nil)
	})
}
