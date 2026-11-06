package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const (
	paginationParametersShapeGoodParams = `        - name: limit
          in: query
          schema: {type: integer, default: 10, maximum: 100}
        - name: offset
          in: query
          schema: {type: integer}`
	paginationParametersShapeLinks = `                  first: {type: object, properties: {href: {type: string}}}
                  next: {type: object, properties: {href: {type: string}}}
                  last: {type: object, properties: {href: {type: string}}}
                  previous: {type: object, properties: {href: {type: string}}}`
	paginationParametersShapeGoodProps = `                  limit: {type: integer}
                  offset: {type: integer}
                  total_count: {type: integer}
` + paginationParametersShapeLinks
	paginationParametersShapeRequiredAll = "limit, offset, total_count, first, next"
	paginationParametersShapeSchema      = "$.paths['/things'].get.responses['200'].content['application/json'].schema"
	paginationParametersShapeProps       = paginationParametersShapeSchema + ".properties."
	paginationParametersShapeFirstParam  = "$.paths['/things'].get.parameters[0]"
)

func paginationParametersShapeDoc(parameters, properties, required string) string {
	return ruletest.Doc30(`
paths:
  /things:
    get:
      parameters:
` + parameters + `
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [` + required + `]
                properties:
                  things:
                    type: array
                    items:
                      type: string
` + properties)
}

func TestPaginationParametersShape(t *testing.T) {
	t.Parallel()

	rule := rules.PaginationParametersShape()

	t.Run("accepts a well formed offset and limit list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule,
			paginationParametersShapeDoc(paginationParametersShapeGoodParams, paginationParametersShapeGoodProps,
				paginationParametersShapeRequiredAll), nil)
	})

	t.Run("flags a required unbounded limit", func(t *testing.T) {
		t.Parallel()

		params := `        - name: limit
          in: query
          required: true
          schema: {type: integer}
        - name: offset
          in: query
          schema: {type: integer}`

		ruletest.RequirePaths(
			t,
			rule,
			paginationParametersShapeDoc(
				params,
				paginationParametersShapeGoodProps,
				paginationParametersShapeRequiredAll,
			),
			nil,
			paginationParametersShapeFirstParam,
		)
	})

	t.Run("treats a zero default as missing and a non integer limit as wrong", func(t *testing.T) {
		t.Parallel()

		params := `        - name: limit
          in: query
          schema: {type: integer, default: 0, maximum: 100}
        - name: offset
          in: query
          schema: {type: integer}`
		ruletest.RequirePaths(
			t,
			rule,
			paginationParametersShapeDoc(
				params,
				paginationParametersShapeGoodProps,
				paginationParametersShapeRequiredAll,
			),
			nil,
			paginationParametersShapeFirstParam,
		)

		params = `        - name: limit
          in: query
          schema: {type: string, default: "10", maximum: 100}
        - name: offset
          in: query
          schema: {type: integer}`
		ruletest.RequirePaths(
			t,
			rule,
			paginationParametersShapeDoc(
				params,
				paginationParametersShapeGoodProps,
				paginationParametersShapeRequiredAll,
			),
			nil,
			paginationParametersShapeFirstParam,
		)
	})

	t.Run("flags a required or non integer offset", func(t *testing.T) {
		t.Parallel()

		params := `        - name: limit
          in: query
          schema: {type: integer, default: 10, maximum: 100}
        - name: offset
          in: query
          required: true
          schema: {type: string}`

		ruletest.RequirePaths(
			t,
			rule,
			paginationParametersShapeDoc(
				params,
				paginationParametersShapeGoodProps,
				paginationParametersShapeRequiredAll,
			),
			nil,
			"$.paths['/things'].get.parameters[1]",
		)
	})

	t.Run("flags offset without limit on the operation", func(t *testing.T) {
		t.Parallel()

		params := `        - name: offset
          in: query
          schema: {type: integer}`
		props := "                  offset: {type: integer}\n" + paginationParametersShapeLinks

		results := ruletest.Run(t, rule,
			paginationParametersShapeDoc(params, props, "offset, first, next"), nil)

		require.Equal(t, []string{"$.paths['/things'].get"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "must define a 'limit' query parameter")
	})

	t.Run("flags page token naming and type", func(t *testing.T) {
		t.Parallel()

		props := paginationParametersShapeLinks
		params := `        - name: cursor
          in: query
          schema: {type: string}`
		results := ruletest.Run(t, rule, paginationParametersShapeDoc(params, props, "first, next"), nil)
		require.Equal(t, []string{paginationParametersShapeFirstParam}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "should be named 'start'")

		params = `        - name: start
          in: query
          required: true
          schema: {type: integer}`
		results = ruletest.Run(t, rule, paginationParametersShapeDoc(params, props, "first, next"), nil)
		require.Equal(t, []string{paginationParametersShapeFirstParam}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "string and optional")

		params = `        - name: start
          in: query
          schema: {type: string}`
		ruletest.RequireClean(t, rule, paginationParametersShapeDoc(params, props, "first, next"), nil)
	})

	t.Run("flags missing and not required response properties", func(t *testing.T) {
		t.Parallel()

		props := `                  offset: {type: integer}
                  total_count: {type: string}
` + paginationParametersShapeLinks

		ruletest.RequirePaths(t, rule,
			paginationParametersShapeDoc(paginationParametersShapeGoodParams, props, "total_count, first, next"), nil,
			paginationParametersShapeSchema,
			paginationParametersShapeProps+"offset",
			paginationParametersShapeProps+"total_count")
	})

	t.Run("flags missing and malformed page links", func(t *testing.T) {
		t.Parallel()

		props := `                  limit: {type: integer}
                  offset: {type: integer}
                  last: {type: string}
                  previous: {type: object, properties: {href: {type: integer}}}`

		ruletest.RequirePaths(t, rule,
			paginationParametersShapeDoc(paginationParametersShapeGoodParams, props, "limit, offset"), nil,
			paginationParametersShapeSchema, paginationParametersShapeSchema,
			paginationParametersShapeProps+"last", paginationParametersShapeProps+"previous")
	})

	t.Run("merges allOf members of the response schema", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /things:
    get:
      parameters:
        - name: start
          in: query
          schema: {type: string}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/Page"
                  - type: object
                    properties:
                      things:
                        type: array
                        items: {type: string}
components:
  schemas:
    Page:
      type: object
      required: [first]
      properties:
        first: {type: object, properties: {href: {type: string}}}
        next: {type: object, properties: {href: {type: integer}}}
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.components.schemas.Page.properties.next")
	})

	t.Run("ignores operations that are not paginated lists", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, paginationParametersShapeDoc(`        - name: q
          in: query
          schema: {type: string}`, "", ""), nil)

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things/{id}:
    get:
      parameters:
        - {name: offset, in: query, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  things: {type: array, items: {type: string}}
`), nil)
	})
}
