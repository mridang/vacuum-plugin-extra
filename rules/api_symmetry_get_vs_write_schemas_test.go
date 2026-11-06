package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const apiSymmetryGetVsWriteSchemasPaths = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
`

func TestAPISymmetryGetVsWriteSchemas(t *testing.T) {
	t.Parallel()

	rule := rules.APISymmetryGetVsWriteSchemas()

	t.Run("flags properties that do not exist on the canonical schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        name: {type: string}
        age: {type: integer}
    PetPrototype:
      type: object
      properties:
        name: {type: string}
        nickname: {type: string}
    PetPatch:
      type: object
      properties:
        age: {type: string}
    PetSummary:
      type: object
      properties:
        name: {type: string}
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.PetPrototype",
			"$.components.schemas.PetPatch",
		}, ruletest.Paths(results))
	})

	t.Run("accepts graph fragments including nested ones", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        name: {type: string}
        owner:
          type: object
          properties:
            id: {type: string}
            email: {type: string}
        tags:
          type: array
          items:
            $ref: "#/components/schemas/Tag"
    Tag:
      type: object
      properties:
        label: {type: string}
    PetPatch:
      type: object
      properties:
        owner:
          type: object
          properties:
            id: {type: string}
        tags:
          type: array
          items:
            type: object
            properties:
              label: {type: string}
`), nil)
	})

	t.Run("flags nested properties that are not fragments", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        owner:
          type: object
          properties:
            id: {type: string}
    PetPatch:
      type: object
      properties:
        owner:
          type: object
          properties:
            phone: {type: string}
`), nil, "$.components.schemas.PetPatch")
	})

	t.Run("flags array items and additionalProperties that are not fragments", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        tags:
          type: array
          items:
            type: object
            properties:
              label: {type: string}
        attrs:
          type: object
          additionalProperties:
            type: string
    PetPatch:
      type: object
      properties:
        tags:
          type: array
          items:
            type: object
            properties:
              colour: {type: string}
    PetSummary:
      type: object
      properties:
        attrs:
          type: object
          additionalProperties:
            type: integer
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.PetPatch",
			"$.components.schemas.PetSummary",
		}, ruletest.Paths(results))
	})

	t.Run("exempts writeOnly properties in prototypes only", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        name: {type: string}
    PetPrototype:
      type: object
      properties:
        name: {type: string}
        password: {type: string, writeOnly: true}
    PetPatch:
      type: object
      properties:
        password: {type: string, writeOnly: true}
`), nil, "$.components.schemas.PetPatch")
	})

	t.Run("checks allOf members of a variant", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      allOf:
        - properties:
            name: {type: string}
    PetPrototype:
      type: object
      allOf:
        - properties:
            name: {type: string}
        - properties:
            extra: {type: string}
`), nil, "$.components.schemas.PetPrototype")
	})

	t.Run("uses the canonical schema in place of a Reference schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(apiSymmetryGetVsWriteSchemasPaths+`
    Pet:
      type: object
      properties:
        owner:
          $ref: "#/components/schemas/PersonReference"
    Person:
      type: object
      properties:
        id: {type: string}
        name: {type: string}
    PersonReference:
      type: object
      properties:
        id: {type: string}
    PetPatch:
      type: object
      properties:
        owner:
          type: object
          properties:
            name: {type: string}
`), nil)
	})

	t.Run("skips resources without a canonical schema reference or sibling path", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string}
    PetPatch:
      type: object
      properties:
        other: {type: string}
`), nil)
	})
}
