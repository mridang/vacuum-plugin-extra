package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestIllogicalCompositionKeywords(t *testing.T) {
	t.Parallel()

	rule := rules.IllogicalCompositionKeywords()

	t.Run("flags single member and empty compositions", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Single:
      oneOf:
        - type: string
    Empty:
      anyOf: []
    Wrapped:
      allOf:
        - type: string
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Single.oneOf",
			"$.components.schemas.Empty.anyOf",
			"$.components.schemas.Wrapped.allOf",
		}, ruletest.Paths(results))
	})

	t.Run("flags duplicate and empty members", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  schemas:
    Dup:
      oneOf:
        - type: string
          maxLength: 3
        - maxLength: 3
          type: string
        - {}
`), nil,
			"$.components.schemas.Dup.oneOf[1]",
			"$.components.schemas.Dup.oneOf[2]",
		)
	})

	t.Run("accepts distinct compositions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Pet:
      oneOf:
        - type: string
        - type: integer
    Described:
      description: extends a reference
      allOf:
        - type: string
`), nil)
	})

	t.Run("accepts a single member with a discriminator", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Base:
      type: object
      discriminator:
        propertyName: kind
    Cat:
      allOf:
        - $ref: "#/components/schemas/Base"
    Pet:
      discriminator:
        propertyName: kind
      oneOf:
        - $ref: "#/components/schemas/Cat"
`), nil)
	})

	t.Run("flags inline members under a discriminator", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    Beverage:
      type: object
      required: [category]
      properties:
        category: {type: string}
    MenuItem:
      discriminator:
        propertyName: category
      oneOf:
        - $ref: "#/components/schemas/Beverage"
        - type: object
          required: [category]
          properties:
            category: {type: string}
`), nil, "$.components.schemas.MenuItem.oneOf[1]")
	})

	t.Run("flags a discriminator property that is not required", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Beverage:
      type: object
      properties:
        category: {type: string, enum: [beverage]}
    Dessert:
      type: object
      properties:
        category: {type: string, enum: [dessert]}
    MenuItem:
      discriminator:
        propertyName: category
      oneOf:
        - $ref: "#/components/schemas/Beverage"
        - $ref: "#/components/schemas/Dessert"
`), nil)

		require.Equal(t, []string{"$.components.schemas.MenuItem.oneOf"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "Add `category` to `required`")
	})

	t.Run("accepts a required discriminator property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Beverage:
      type: object
      required: [category]
      properties:
        category: {type: string, enum: [beverage]}
    Dessert:
      type: object
      required: [category]
      properties:
        category: {type: string, enum: [dessert]}
    MenuItem:
      discriminator:
        propertyName: category
      oneOf:
        - $ref: "#/components/schemas/Beverage"
        - $ref: "#/components/schemas/Dessert"
`), nil)
	})

	t.Run("leaves the discriminator gap to the 3.2 rule", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, `
openapi: 3.2.0
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    A:
      type: object
      properties:
        kind: {type: string}
    B:
      type: object
      properties:
        kind: {type: string}
    C:
      discriminator:
        propertyName: kind
      oneOf:
        - $ref: "#/components/schemas/A"
        - $ref: "#/components/schemas/B"
`, nil)
	})

	t.Run("flags members that both accept null", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    PhotoUrl:
      type: [string, "null"]
    Photo:
      oneOf:
        - $ref: "#/components/schemas/PhotoUrl"
        - type: "null"
`), nil)

		require.Equal(t, []string{"$.components.schemas.Photo.oneOf"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "Both schemas accept `null`")
	})

	t.Run("flags a nullable parent with a nullable member", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Photo:
      nullable: true
      oneOf:
        - type: string
          nullable: true
        - type: integer
`), nil)

		require.Contains(
			t,
			ruletest.Messages(results),
			"The schema and a schema in `oneOf` both accept `null`, so nothing decides which one applies to a null value.",
		)
	})

	t.Run("flags overlapping types including integer against number", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Same:
      oneOf:
        - type: string
        - type: [string, integer]
    Num:
      oneOf:
        - type: integer
        - type: number
    Distinct:
      oneOf:
        - type: string
        - type: integer
`), nil)

		require.ElementsMatch(t, []string{
			"$.components.schemas.Same.oneOf",
			"$.components.schemas.Num.oneOf",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results)[0]+ruletest.Messages(results)[1],
			"Both schemas accept `integer`.",
		)
	})

	t.Run("compares const against enum", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc31(`
components:
  schemas:
    Kind:
      oneOf:
        - const: beverage
        - enum: [beverage, dessert]
    Other:
      oneOf:
        - const: beverage
        - enum: [dessert]
`), nil)

		require.Equal(t, []string{"$.components.schemas.Kind.oneOf"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], `Both schemas allow the values ["beverage"].`)
	})

	t.Run("flags shared properties without exclusive constraints", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Shared:
      oneOf:
        - type: object
          properties:
            name: {type: string}
        - type: object
          properties:
            name: {type: string}
            extra: {type: string}
`), nil)

		require.Equal(t, []string{"$.components.schemas.Shared.oneOf"}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results)[0],
			"Both schemas define `name` without constraints that exclude each other.",
		)
	})

	t.Run("flags optional distinguishing properties and accepts required ones", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Optional:
      oneOf:
        - type: object
          properties:
            category: {type: string, enum: [a]}
        - type: object
          properties:
            category: {type: string, enum: [b]}
    Required:
      oneOf:
        - type: object
          required: [category]
          properties:
            category: {type: string, enum: [a]}
        - type: object
          required: [category]
          properties:
            category: {type: string, enum: [b]}
`), nil)

		require.Equal(t, []string{"$.components.schemas.Optional.oneOf"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "an optional property cannot distinguish the schemas")
	})

	t.Run("ignores pairs that use constraints it does not model", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Patterns:
      oneOf:
        - type: string
          pattern: "^a"
        - type: string
          pattern: "^b"
`), nil)
	})

	t.Run("accepts members that require different properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
components:
  schemas:
    Either:
      oneOf:
        - type: object
          required: [a]
          properties:
            a: {type: string}
            shared: {type: string}
        - type: object
          required: [b]
          properties:
            b: {type: string}
            shared: {type: string}
`), nil)
	})

	t.Run("reports each overlapping pair", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Three:
      oneOf:
        - type: string
        - type: string
          title: Second
        - type: string
          title: Third
`), nil)

		require.Len(t, results, 3)
	})

	t.Run("flags an anyOf with a discriminator and an inline member", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
components:
  schemas:
    A:
      type: object
    Pet:
      discriminator:
        propertyName: kind
      anyOf:
        - $ref: "#/components/schemas/A"
        - type: object
`), nil, "$.components.schemas.Pet.anyOf[1]")
	})

	t.Run("names the earlier position of a duplicate", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
components:
  schemas:
    Dup:
      allOf:
        - type: string
        - type: integer
        - type: string
`), nil)

		require.Equal(t, []string{"$.components.schemas.Dup.allOf[2]"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "duplicates the schema at position 1")
	})
}
