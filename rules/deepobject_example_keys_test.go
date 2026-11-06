package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDeepobjectExampleKeys(t *testing.T) {
	t.Parallel()

	rule := rules.DeepobjectExampleKeys()

	t.Run("flags unknown top-level and nested keys", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          style: deepObject
          schema:
            $ref: "#/components/schemas/Filter"
          example:
            state: open
            owner:
              nme: bob
          examples:
            one:
              value:
                status: open
                bogus: 1
      responses:
        "200":
          description: ok
components:
  schemas:
    Filter:
      type: object
      properties:
        status: {type: string}
        owner:
          type: object
          properties:
            name: {type: string}
`), nil,
			"$.paths['/pets'].get.parameters[0].example.state",
			"$.paths['/pets'].get.parameters[0].example.owner.nme",
			"$.paths['/pets'].get.parameters[0].examples.one.value.bogus",
		)
	})

	t.Run("accepts matching keys and open schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          style: deepObject
          schema:
            type: object
            properties:
              status: {type: string}
          example:
            status: open
        - name: meta
          in: query
          style: deepObject
          schema:
            type: object
            additionalProperties: {type: string}
          example:
            anything: x
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("ignores parameters that are not deepObject", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: filter
          in: query
          schema:
            type: object
            properties:
              status: {type: string}
          example:
            state: open
      responses:
        "200":
          description: ok
`), nil)
	})
}
