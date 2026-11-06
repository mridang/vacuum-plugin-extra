package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const preferTokenPaginationOverOffsetBody = `
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  things: {type: array, items: {type: string}}
`

func TestPreferTokenPaginationOverOffset(t *testing.T) {
	t.Parallel()

	rule := rules.PreferTokenPaginationOverOffset()

	t.Run("flags offset on paginated list operations at the operation", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
        - {name: offset, in: query, schema: {type: integer}}
`+preferTokenPaginationOverOffsetBody+`
  /owners:
    get:
      parameters:
        - {name: offset, in: query, schema: {type: integer}}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/Page"
components:
  schemas:
    Page:
      type: object
      properties:
        owners:
          type: array
          items: {type: string}
`), nil)

		require.ElementsMatch(t, []string{"$.paths['/pets'].get", "$.paths['/owners'].get"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "token-based pagination is recommended")
	})

	t.Run("accepts token pagination", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
        - {name: start, in: query, schema: {type: string}}
`+preferTokenPaginationOverOffsetBody), nil)
	})

	t.Run("ignores operations that are not paginated lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: offset, in: query, schema: {type: integer}}
      responses:
        "200":
          description: ok
  /pets/{id}:
    get:
      parameters:
        - {name: offset, in: query, schema: {type: integer}}
`+preferTokenPaginationOverOffsetBody+`
  /owners:
    get:
      parameters:
        - {name: limit, in: query, schema: {type: integer}}
`+preferTokenPaginationOverOffsetBody), nil)
	})

	t.Run("matches the offset name exactly and only in the query", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - {name: Offset, in: query, schema: {type: integer}}
        - {name: offset, in: header, schema: {type: integer}}
        - {name: start, in: query, schema: {type: string}}
`+preferTokenPaginationOverOffsetBody), nil)
	})
}
