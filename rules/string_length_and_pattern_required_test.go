package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/formats"
	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestStringLengthAndPatternRequired(t *testing.T) {
	t.Parallel()

	rule := rules.StringLengthAndPatternRequired()

	t.Run("flags each missing keyword of nested properties", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Thing:
      type: object
      properties:
        name:
          type: string
        partial:
          type: string
          pattern: "^a"
          minLength: 1
`), nil)

		require.Equal(t, []string{
			"$.components.schemas.Thing.properties.name",
			"$.components.schemas.Thing.properties.name",
			"$.components.schemas.Thing.properties.name",
			"$.components.schemas.Thing.properties.partial",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results), "String schemas should define property 'maxLength'")
	})

	t.Run("applies the IBM format bypasses per keyword", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    DateTime:
      type: string
      format: date-time
    Date:
      type: string
      format: date
    Ident:
      type: string
      format: identifier
      pattern: "^a"
      maxLength: 4
    Url:
      type: string
      format: url
      maxLength: 4
`), nil)

		require.Equal(t, []string{"$.components.schemas.DateTime"}, ruletest.Paths(results))
		require.Equal(t, []string{"String schemas should define property 'maxLength'"}, ruletest.Messages(results))
	})

	t.Run("bypasses everything for enums, also through allOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Kind:
      type: string
      enum: [a, b]
    Base:
      enum: [a]
    Composed:
      type: string
      allOf:
        - $ref: "#/components/schemas/Base"
`), nil)
	})

	t.Run("bypasses everything for const", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
components:
  schemas:
    Error:
      type: object
      properties:
        code:
          type: string
          const: not_found
`), nil)
	})

	t.Run("reads attributes through allOf and oneOf", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Bounds:
      pattern: "^a"
      minLength: 1
      maxLength: 3
    Name:
      type: string
      allOf:
        - $ref: "#/components/schemas/Bounds"
    Either:
      type: string
      oneOf:
        - {pattern: "^a", minLength: 1, maxLength: 2}
        - {pattern: "^b", minLength: 1, maxLength: 2}
`), nil)
	})

	t.Run("flags minLength greater than maxLength", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Name:
      type: string
      pattern: "^a"
      minLength: 9
      maxLength: 3
`), nil)

		require.Equal(t, []string{"'minLength' cannot be greater than 'maxLength'"}, ruletest.Messages(results))
	})

	t.Run("flags string keywords on non-string schemas", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Count:
      type: integer
      minLength: 1
      pattern: "^1"
`), nil,
			"$.components.schemas.Count.pattern",
			"$.components.schemas.Count.minLength",
		)
	})

	t.Run("honours the require options", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc31(`
components:
  schemas:
    Text:
      type: [string, "null"]
      minLength: 1
      maxLength: 10
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.components.schemas.Text")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"requirePattern": false})
	})

	t.Run("skips pattern and minLength for every shared format but keeps maxLength", func(t *testing.T) {
		t.Parallel()

		for _, format := range formats.NonEmpty() {
			t.Run(format, func(t *testing.T) {
				t.Parallel()

				results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Thing:
      type: object
      properties:
        value:
          type: string
          format: `+format+`
`), nil)
				without := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Thing:
      type: object
      properties:
        value:
          type: string
`), nil)

				require.Len(t, without, 3)
				require.NotContains(t, ruletest.Messages(results), "String schemas should define property 'pattern'")
				require.NotContains(t, ruletest.Messages(results), "String schemas should define property 'minLength'")
			})
		}
	})

	t.Run("honours the shapedFormats option", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc31(`
components:
  schemas:
    Thing:
      type: object
      properties:
        value:
          type: string
          format: email
          maxLength: 64
`)

		ruletest.RequireClean(t, rule, document, nil)
		require.Len(t, ruletest.Run(t, rule, document, ruletest.Options{"shapedFormats": []any{}}), 2)
	})
}
