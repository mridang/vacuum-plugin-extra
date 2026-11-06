package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestDateTimePropertySuffix(t *testing.T) {
	t.Parallel()

	rule := rules.DateTimePropertySuffix()

	t.Run("flags properties whose names do not match", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        created:
          type: string
          format: date-time
        birthday:
          type: string
          format: date
        opens:
          type: string
          format: time
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Pet.properties.created",
			"$.components.schemas.Pet.properties.birthday",
			"$.components.schemas.Pet.properties.opens",
		}, ruletest.Paths(results))
		require.Contains(t, strings.Join(ruletest.Messages(results), "\n"), "format \"date-time\"")
	})

	t.Run("accepts names matching one of the default patterns", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        created_at:
          type: string
          format: date-time
        last_timestamp:
          type: string
          format: date-time
        birth_date:
          type: string
          format: date
        closing_time:
          type: string
          format: time
        name:
          type: string
`), nil)
	})

	t.Run("follows a reference to the format", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Stamp:
      type: string
      format: date-time
    Pet:
      type: object
      properties:
        created:
          $ref: "#/components/schemas/Stamp"
`), nil, "$.components.schemas.Pet.properties.created")
	})

	t.Run("reports camel case names the original patterns reject", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        createdAt:
          type: string
          format: date-time
`), nil, "$.components.schemas.Pet.properties.createdAt")
	})

	t.Run("requires a full match", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      type: object
      properties:
        created_at_x:
          type: string
          format: date-time
`), nil, "$.components.schemas.Pet.properties.created_at_x")
	})

	t.Run("honours option overrides", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
components:
  schemas:
    Pet:
      type: object
      properties:
        created:
          type: string
          format: date-time
`)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"patterns": []string{"created"}})
		ruletest.RequireClean(t, rule, document, ruletest.Options{"patterns": []string{}})
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"patterns": []string{"_on"}},
			"$.components.schemas.Pet.properties.created")
	})
}
