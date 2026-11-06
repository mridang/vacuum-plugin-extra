package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestSchemaTypeFormatWhitelist(t *testing.T) {
	t.Parallel()

	rule := rules.SchemaTypeFormatWhitelist()

	t.Run("flags unknown types and formats", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Count:
      type: int
    When:
      type: string
      format: datetime
    Big:
      type: integer
      format: int128
    Flag:
      type: boolean
      format: yesno
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Count.type",
			"$.components.schemas.When.format",
			"$.components.schemas.Big.format",
			"$.components.schemas.Flag.format",
		}, ruletest.Paths(results))
	})

	t.Run("accepts the JSON Schema 2020-12 string formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Ttl:
      type: string
      format: duration
    Clock:
      type: string
      format: time
`), nil)
	})

	t.Run("accepts standard types and formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: integer
      format: int64
    B:
      type: number
      format: double
    C:
      type: string
      format: date-time
    D:
      type: object
`), nil)
	})

	t.Run("handles 3.1 type lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: [string, "null"]
      format: uuid
`), nil)

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    A:
      type: [string, "null"]
      format: nope
`), nil, "$.components.schemas.A.format")
	})

	t.Run("honours option overrides", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
components:
  schemas:
    A:
      type: string
      format: money
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.components.schemas.A.format")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"stringFormats": []any{"money"}})
	})
}
