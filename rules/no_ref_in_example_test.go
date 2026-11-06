package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoRefInExample(t *testing.T) {
	t.Parallel()

	rule := rules.NoRefInExample()

	t.Run("flags a ref inside an example", func(t *testing.T) {
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
              example:
                $ref: "#/components/examples/Pet"
components:
  examples:
    Pet:
      value:
        name: Rex
`), nil)

		require.Len(t, results, 1)
		require.Equal(
			t,
			"$.paths['/pets'].get.responses['200'].content['application/json'].example['$ref']",
			results[0].Path,
		)
	})

	t.Run("accepts a ref inside examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              examples:
                pet:
                  $ref: "#/components/examples/Pet"
components:
  examples:
    Pet:
      value:
        name: Rex
`), nil)
	})

	t.Run("finds nested refs in schema and parameter examples", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - name: q
          in: query
          example:
            - $ref: "#/components/examples/Pet"
          schema:
            type: string
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
      example:
        owner:
          $ref: "#/components/schemas/Owner"
`), nil,
			"$.paths['/pets'].get.parameters[0].example[0]['$ref']",
			"$.components.schemas.Pet.example.owner['$ref']",
		)
	})

	t.Run("does not mistake a property called example for the keyword", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Doc:
      type: object
      properties:
        example:
          $ref: "#/components/schemas/Sample"
    Sample:
      type: string
`), nil)
	})
}
