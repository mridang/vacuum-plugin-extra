package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSuccessResponseExample(t *testing.T) {
	t.Parallel()

	rule := rules.SuccessResponseExample()

	t.Run("flags a success response without an example", func(t *testing.T) {
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
              schema:
                type: object
        "404":
          description: missing
          content:
            application/json:
              schema:
                type: object
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/pets'].get.responses['200'].content['application/json']", results[0].Path)
		require.Contains(t, results[0].Message, "example")
	})

	t.Run("accepts examples on the media type or on the schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
              example: {name: Rex}
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    put:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                one:
                  value: {name: Rex}
components:
  schemas:
    Pet:
      type: object
      example: {name: Rex}
`), nil)
	})

	t.Run("checks only the configured media types", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            text/plain:
              schema:
                type: string
`)

		ruletest.RequireClean(t, rule, document, nil)
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"mediaTypes": []string{"text/plain"}},
			"$.paths['/pets'].get.responses['200'].content['text/plain']")
	})

	t.Run("ignores responses without content", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    delete:
      responses:
        "204":
          description: gone
`), nil)
	})
}
