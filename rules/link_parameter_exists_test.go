package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const linkParameterExistsBase = `
paths:
  /pets/{id}:
    get:
      operationId: getPet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
        - {name: id, in: query, schema: {type: string}}
        - {name: fields, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
  /owners:
    get:
      responses:
        "200":
          description: ok
          links:
`

func TestLinkParameterExists(t *testing.T) {
	t.Parallel()

	rule := rules.LinkParameterExists()

	t.Run("flags an unknown parameter via operationId", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            pet:
              operationId: getPet
              parameters:
                petId: $response.body#/id
                fields: x
                query.id: y
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/owners'].get.responses['200'].links.pet.parameters.petId", results[0].Path)
	})

	t.Run("flags an unknown parameter via operationRef", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            pet:
              operationRef: "#/paths/~1pets~1{id}/get"
              parameters:
                path.nope: x
`), nil, "$.paths['/owners'].get.responses['200'].links.pet.parameters['path.nope']")
	})

	t.Run("warns about an ambiguous bare name", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            pet:
              operationId: getPet
              parameters:
                id: x
`), nil)

		require.Len(t, results, 1)
		require.Contains(t, results[0].Message, "ambiguous")
	})

	t.Run("checks component links", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets/{id}:
    get:
      operationId: getPet
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      responses:
        "200":
          description: ok
components:
  links:
    Pet:
      operationId: getPet
      parameters:
        wrong: x
`), nil, "$.components.links.Pet.parameters.wrong")
	})

	t.Run("does not check parameters of unresolved targets", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            pet:
              operationId: missing
              parameters:
                anything: x
`), nil, "$.paths['/owners'].get.responses['200'].links.pet.operationId")
	})

	t.Run("requires exactly one of operationId and operationRef", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            neither:
              description: nothing
            both:
              operationId: getPet
              operationRef: "#/paths/~1pets~1{id}/get"
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/owners'].get.responses['200'].links.neither",
			"$.paths['/owners'].get.responses['200'].links.both",
		}, ruletest.Paths(results))
	})

	t.Run("flags unknown operations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            byId:
              operationId: nope
            byRef:
              operationRef: "#/paths/~1missing/get"
`), nil,
			"$.paths['/owners'].get.responses['200'].links.byId.operationId",
			"$.paths['/owners'].get.responses['200'].links.byRef.operationRef",
		)
	})

	t.Run("decodes a percent-encoded operationRef", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(linkParameterExistsBase+`
            pet:
              operationRef: "#/paths/~1pets~1%7Bid%7D/get"
              parameters:
                nope: x
`), nil, "$.paths['/owners'].get.responses['200'].links.pet.parameters.nope")
	})

	t.Run("reports remote operationRef unless switched off", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(linkParameterExistsBase + `
            remote:
              operationRef: "other.yaml#/paths/~1pets/get"
`)

		ruletest.RequirePaths(t, rule, doc, nil, "$.paths['/owners'].get.responses['200'].links.remote.operationRef")
		ruletest.RequireClean(t, rule, doc, ruletest.Options{"reportRemoteOperationRef": false})
	})
}
