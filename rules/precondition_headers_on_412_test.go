package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestPreconditionHeadersOn412(t *testing.T) {
	t.Parallel()

	rule := rules.PreconditionHeadersOn412()

	t.Run("flags a 412 response without a conditional header", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      parameters:
        - name: X-Other
          in: header
          schema:
            type: string
      responses:
        "412":
          description: precondition failed
`), nil)

		require.Equal(t, []string{"$.paths['/pets/{id}'].put.responses['412']"}, ruletest.Paths(results))
	})

	t.Run("accepts an operation header regardless of case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      parameters:
        - name: if-match
          in: header
          schema:
            type: string
      responses:
        "412":
          description: precondition failed
`), nil)
	})

	t.Run("accepts a path level and a referenced header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets/{id}:
    parameters:
      - $ref: "#/components/parameters/IfNoneMatch"
    put:
      responses:
        "412":
          description: precondition failed
components:
  parameters:
    IfNoneMatch:
      name: If-None-Match
      in: header
      schema:
        type: string
`), nil)
	})

	t.Run("ignores a query parameter with the same name", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      parameters:
        - name: If-Match
          in: query
          schema:
            type: string
      responses:
        "412":
          description: precondition failed
`), nil, "$.paths['/pets/{id}'].put.responses['412']")
	})

	t.Run("ignores operations without a 412 response", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("honours the headers option", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      parameters:
        - name: X-Version
          in: header
          schema:
            type: string
      responses:
        "412":
          description: precondition failed
`), ruletest.Options{"headers": []string{"X-Version"}})
	})
}
