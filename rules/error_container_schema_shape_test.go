package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const errorContainerSchemaShapeValid = `
      type: object
      properties:
        trace:
          type: string
        status_code:
          type: integer
        errors:
          type: array
          items:
            type: object
            properties:
              code:
                type: string
                enum: [not_found]
              message:
                type: string
              more_info:
                type: string
              target:
                type: object
                properties:
                  name:
                    type: string
                  type:
                    type: string
                    enum: [parameter, field, header]
`

func errorContainerSchemaShapeDoc(schema string) string {
	return ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
        "404":
          description: not found
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
components:
  schemas:
    Error:
` + schema)
}

func TestErrorContainerSchemaShape(t *testing.T) {
	t.Parallel()

	rule := rules.ErrorContainerSchemaShape()

	t.Run("accepts a valid error container", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, errorContainerSchemaShapeDoc(errorContainerSchemaShapeValid), nil)
	})

	t.Run("flags a schema that is not an object with properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, errorContainerSchemaShapeDoc(`
      type: string
`), nil, "$.components.schemas.Error")
	})

	t.Run("flags missing trace and errors", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, errorContainerSchemaShapeDoc(`
      type: object
      properties:
        message:
          type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Error.properties",
			"$.components.schemas.Error.properties",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"Error container model should contain property 'trace' of type string",
		)
	})

	t.Run("flags a bad error model and target", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, errorContainerSchemaShapeDoc(`
      type: object
      properties:
        trace:
          type: string
        errors:
          type: array
          items:
            type: object
            properties:
              code:
                type: string
              message:
                type: integer
              more_info:
                type: string
              target:
                type: object
                properties:
                  name:
                    type: string
                  type:
                    type: string
                    enum: [field]
`), nil,
			"$.components.schemas.Error.properties.errors.items.properties.code",
			"$.components.schemas.Error.properties.errors.items.properties.message",
			"$.components.schemas.Error.properties.errors.items.properties.target.properties.type.enum",
		)
	})

	t.Run("ignores success responses, ranges and default", func(t *testing.T) {
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
              schema:
                type: string
        "5XX":
          description: boom
          content:
            application/json:
              schema:
                type: string
        default:
          description: boom
          content:
            application/json:
              schema:
                type: string
`), nil)
	})

	t.Run("checks every media type, not only JSON", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "500":
          description: boom
          content:
            text/plain:
              schema:
                type: string
`), nil, "$.paths['/pets'].get.responses['500'].content['text/plain'].schema")
	})

	t.Run("uses the IBM messages and locations", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, errorContainerSchemaShapeDoc(`
      type: object
      properties:
        trace:
          type: integer
        status_code:
          type: string
        errors:
          type: array
          items:
            type: object
            properties:
              code:
                type: string
                enum: not-a-list
              more_info:
                type: string
              target:
                type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Error.properties.trace",
			"$.components.schemas.Error.properties.status_code",
			"$.components.schemas.Error.properties.errors.items.properties.code.enum",
			"$.components.schemas.Error.properties.errors.items.properties",
			"$.components.schemas.Error.properties.errors.items.properties.target",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"Error container model should contain property 'trace' of type string",
		)
		require.Contains(
			t,
			ruletest.Messages(results),
			"Error container model property 'status_code' must be of type integer",
		)
		require.Contains(t, ruletest.Messages(results), "Error model must contain property 'message' of type string")
		require.Contains(
			t,
			ruletest.Messages(results),
			"Error model property 'target' must be a valid error target model object",
		)
	})

	t.Run("accepts the target type enum in any order and rejects duplicates", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, errorContainerSchemaShapeDoc(`
      type: object
      properties:
        trace:
          type: string
        errors:
          type: array
          items:
            type: object
            properties:
              code:
                type: string
                enum: [a]
              message:
                type: string
              more_info:
                type: string
              target:
                type: object
                properties:
                  name:
                    type: integer
                  type:
                    type: string
                    enum: [header, header, field]
`), nil,
			"$.components.schemas.Error.properties.errors.items.properties.target.properties.type.enum",
			"$.components.schemas.Error.properties.errors.items.properties.target.properties.name")
	})
}
