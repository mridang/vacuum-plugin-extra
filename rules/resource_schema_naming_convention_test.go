package rules_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const resourceSchemaNamingConventionGood = `
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PetCollection"
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PetPrototype"
      responses:
        "201":
          description: created
  /pets/{pet_id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    put:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
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
    Pet:
      type: object
    PetPrototype:
      type: object
    PetPatch:
      type: object
    PetCollection:
      type: object
      properties:
        pets:
          type: array
          items:
            $ref: "#/components/schemas/PetSummary"
    PetSummary:
      type: object
`

func TestResourceSchemaNamingConvention(t *testing.T) {
	t.Parallel()

	rule := rules.ResourceSchemaNamingConvention()

	t.Run("accepts conforming names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(resourceSchemaNamingConventionGood), nil)
	})

	t.Run("flags collection, prototype and patch names", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PetList"
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/NewPet"
      responses:
        "201":
          description: created
  /pets/{pet_id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
    patch:
      requestBody:
        content:
          application/json-patch+json:
            schema:
              $ref: "#/components/schemas/PetUpdate"
      responses:
        "200":
          description: ok
components:
  schemas:
    Pet:
      type: object
    PetList:
      type: object
    NewPet:
      type: object
    PetUpdate:
      type: object
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/pets'].get.responses['200'].content['application/json'].schema",
			"$.paths['/pets'].post.requestBody.content['application/json'].schema",
			"$.paths['/pets/{pet_id}'].patch.requestBody.content['application/json-patch+json'].schema",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "PetCollection")
	})

	t.Run("flags the items schema of the collection property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/PetCollection"
  /pets/{pet_id}:
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
    PetCollection:
      type: object
      properties:
        pets:
          type: array
          items:
            $ref: "#/components/schemas/Animal"
    Animal:
      type: object
`), nil, "$.components.schemas.PetCollection.properties.pets.items")
	})

	t.Run("ignores paths without a sibling instance path", func(t *testing.T) {
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
                $ref: "#/components/schemas/Whatever"
components:
  schemas:
    Whatever:
      type: object
`), nil)
	})

	t.Run("derives the names in the configured casing", func(t *testing.T) {
		t.Parallel()

		snake := strings.NewReplacer(
			"PetCollection", "Pet_collection",
			"PetPrototype", "Pet_prototype",
			"PetPatch", "Pet_patch",
			"PetSummary", "Pet_summary",
		).Replace(resourceSchemaNamingConventionGood)
		options := ruletest.Options{"casing": "snake_case"}

		ruletest.RequireClean(t, rule, ruletest.Doc30(snake), options)
		require.NotEmpty(t, ruletest.Run(t, rule, ruletest.Doc30(snake), nil))
		require.Contains(t,
			ruletest.Messages(ruletest.Run(t, rule, ruletest.Doc30(resourceSchemaNamingConventionGood), options)),
			"Collection schema for path '/pets' should use the name: Pet_collection")
	})

	t.Run("disables the check for an empty casing list and reports an unknown one", func(t *testing.T) {
		t.Parallel()

		document := ruletest.Doc30(resourceSchemaNamingConventionGood)

		ruletest.RequireClean(t, rule, document, ruletest.Options{"casing": []string{}})
		ruletest.RequirePaths(t, rule, document, ruletest.Options{"casing": "bogus"}, "$")
	})
}
