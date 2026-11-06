package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDateNamedPropertyMustUseDateFormat(t *testing.T) {
	t.Parallel()

	rule := rules.DateNamedPropertyMustUseDateFormat()

	t.Run("flags date-like names without a date format", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        created_at:
          type: string
        birth_date:
          type: string
          format: date
        expires:
          type: integer
        depends_on:
          type: string
        name:
          type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet.properties.created_at",
			"$.components.schemas.Pet.properties.expires",
		}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "date")
	})

	t.Run("accepts date and date-time formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        created_at:
          type: string
          format: date-time
        updated_timestamp:
          type: string
          format: date
`), nil)
	})

	t.Run("flags a date example without a format", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        birthday:
          type: string
          example: "2020-05-17"
        nickname:
          type: string
          example: Rex
`), nil, "$.components.schemas.Pet.properties.birthday")
	})

	t.Run("honours the names option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        birthday:
          type: string
`)

		ruletest.RequireClean(t, rule, document, nil)
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"names": []any{"birthday"}},
			"$.components.schemas.Pet.properties.birthday")
	})

	t.Run("resolves property references", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Stamp:
      type: string
    Pet:
      type: object
      properties:
        created_at:
          $ref: "#/components/schemas/Stamp"
`), nil, "$.components.schemas.Pet.properties.created_at")
	})

	t.Run("reads examples from the parent schema, media type and parameter", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: since
          in: query
          example: "2020-05-17"
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              example:
                note: "2020-05-17T10:00:00Z"
              schema:
                type: object
                properties:
                  note:
                    type: string
                  other:
                    type: string
components:
  schemas:
    Pet:
      type: object
      example:
        born: "2020-05-17"
        list: ["2020-05-17"]
      properties:
        born:
          type: string
        list:
          type: array
          items:
            type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.parameters[0].schema",
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema.properties.note",
			"$.components.schemas.Pet.properties.born",
			"$.components.schemas.Pet.properties.list.items",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			`According to its example value, this schema should use type "string" and format "date" or "date-time"`,
		)
	})

	t.Run("ignores free text examples and non-date strings", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        text:
          type: string
          example: "see 2020-05-17 for details"
        mac:
          type: string
          example: "aa:bb:cc:dd:ee:ff"
        day:
          type: string
          example: "Monday"
`), nil)
	})

	t.Run("accepts composition that supplies a date format", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Stamp:
      type: string
      format: date-time
    Pet:
      type: object
      properties:
        created_at:
          allOf:
            - $ref: "#/components/schemas/Stamp"
`), nil)
	})

	t.Run("uses the IBM name message", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        created_at:
          type: integer
`), nil)

		require.Equal(
			t,
			[]string{`According to its name, this property should use type "string" and format "date" or "date-time"`},
			ruletest.Messages(results),
		)
	})
}
