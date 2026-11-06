package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestExampleValueXorExternal(t *testing.T) {
	t.Parallel()

	rule := rules.ExampleValueXorExternal()

	t.Run("flags an example with both value and externalValue", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                both:
                  value: {name: Rex}
                  externalValue: https://example.org/rex.json
`), nil)

		require.Len(t, results, 1)
		require.Equal(
			t,
			"$.paths['/pets'].get.responses['200'].content['application/json'].examples.both",
			results[0].Path,
		)
		require.Contains(t, results[0].Message, "both")
	})

	t.Run("flags an example with neither keyword", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  examples:
    Empty:
      summary: nothing here
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.examples.Empty", results[0].Path)
		require.Contains(t, results[0].Message, "either")
	})

	t.Run("accepts exactly one keyword everywhere examples can appear", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    post:
      parameters:
        - name: q
          in: query
          schema:
            type: string
          examples:
            a:
              value: x
      requestBody:
        content:
          application/json:
            examples:
              a:
                externalValue: https://example.org/a.json
      responses:
        "200":
          description: ok
          headers:
            X-Rate:
              schema:
                type: integer
              examples:
                a:
                  value: 1
`), nil)
	})

	t.Run("does not look into schema properties named examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Doc:
      type: object
      properties:
        examples:
          type: array
          items:
            type: string
`), nil)
	})

	t.Run("resolves referenced examples once", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                a:
                  $ref: "#/components/examples/Both"
components:
  examples:
    Both:
      value: 1
      externalValue: https://example.org/1.json
`), nil, "$.components.examples.Both")
	})

	t.Run("honours the options", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
components:
  examples:
    Both:
      value: 1
      externalValue: https://example.org/1.json
    Neither:
      summary: none
`)

		ruletest.RequirePaths(t, rule, document, ruletest.Options{"requireOne": false}, "$.components.examples.Both")
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"forbidBoth": false}, "$.components.examples.Neither")
	})
}
