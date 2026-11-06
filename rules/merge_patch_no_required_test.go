package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestMergePatchNoRequired(t *testing.T) {
	t.Parallel()

	rule := rules.MergePatchNoRequired()

	t.Run("flags required in an inline merge patch schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
                owner:
                  type: object
                  required: [id]
                  properties:
                    id:
                      type: string
      responses:
        "200":
          description: ok
`), nil)

		base := "$.paths['/pets/{id}'].patch.requestBody.content['application/merge-patch+json'].schema"
		require.ElementsMatch(t, []string{
			base + ".required",
			base + ".properties.owner.required",
		}, ruletest.Paths(results))
	})

	t.Run("reports a referenced schema on the request body schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              $ref: "#/components/schemas/PetPatch"
      responses:
        "200":
          description: ok
components:
  schemas:
    PetPatch:
      type: object
      required: [name]
      properties:
        name:
          type: string
`), nil, "$.paths['/pets/{id}'].patch.requestBody.content['application/merge-patch+json'].schema")
	})

	t.Run("accepts a merge patch schema without required", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
              required: []
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags a positive minProperties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    patch:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
              minProperties: 1
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets/{id}'].patch.requestBody.content['application/merge-patch+json'].schema.minProperties")
	})

	t.Run("ignores other media types and methods", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    put:
      requestBody:
        content:
          application/merge-patch+json:
            schema:
              type: object
              required: [name]
      responses:
        "200":
          description: ok
    patch:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
      responses:
        "200":
          description: ok
`), nil)
	})
}
