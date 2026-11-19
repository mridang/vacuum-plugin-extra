package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const resourceResponseConsistencySchemas = `
components:
  schemas:
    Thing:
      type: object
      properties:
        id: {type: string}
        name: {type: string}
    ThingSummary:
      type: object
      properties:
        id: {type: string}
`

func TestResourceResponseConsistency(t *testing.T) {
	t.Parallel()

	rule := rules.ResourceResponseConsistency()

	t.Run("flags a create that returns a different schema than the GET", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/things'].post.responses['201'].content['application/json']", results[0].Path)
		require.Equal(
			t,
			"Operations on a single resource instance should return the resource canonical schema: Thing",
			results[0].Message,
		)
	})

	t.Run("flags PUT and PATCH on the instance path", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
    put:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
    patch:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things/{id}'].put.responses['200'].content['application/json']")
	})

	t.Run("compares a bulk PUT with the collection GET", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Thing"
    put:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things'].put.responses['200'].content['application/json']")
	})

	t.Run("accepts identical schemas inline or by reference", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      operationId: createThing
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  name: {type: string}
                  id: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("skips operations that are not creates and resources without a GET", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /search:
    post:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
  /things/{id}:
    put:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("uses the collection message for a bulk PUT on a Collection schema", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingCollection"
    put:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
components:
  schemas:
    Thing:
      type: object
    ThingCollection:
      type: object
      properties:
        things:
          type: array
          items: {$ref: "#/components/schemas/Thing"}
`), nil)

		require.Equal(
			t,
			[]string{"$.paths['/things'].put.responses['200'].content['application/json']"},
			ruletest.Paths(results),
		)
		require.Equal(t,
			[]string{"Bulk resource operations should return the resource collection schema: ThingCollection"},
			ruletest.Messages(results))
	})

	t.Run("falls back to the first 2xx JSON schema when the GET has no 200", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things/{id}:
    get:
      responses:
        "203":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
    patch:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
        "201":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things/{id}'].patch.responses['200'].content['application/json']")
	})

	t.Run("ignores non application/json media types and create POSTs without a sibling", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/vnd.api+json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
  /other:
    post:
      operationId: createOther
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingSummary"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("ignores writeOnly properties on both sides", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                required: [id, name]
                properties:
                  id: {type: string}
                  name: {type: string}
                  secret: {type: string, writeOnly: true}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [id, name, token]
                properties:
                  id: {type: string}
                  name: {type: string}
                  token: {type: string, writeOnly: true}
`), nil)
	})

	t.Run("accepts an allOf member whose required list is emptied by writeOnly properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                allOf:
                  - type: object
                    required: [secret]
                    properties:
                      id: {type: string}
                      secret: {type: string, writeOnly: true}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
`), nil)
	})

	t.Run("still reports an unmarked difference beside writeOnly properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  name: {type: integer}
                  secret: {type: string, writeOnly: true}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  name: {type: string}
`), nil, "$.paths['/things'].post.responses['201'].content['application/json']")
	})

	t.Run("accepts a create whose allOf includes the GET schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/Thing"
                  - type: object
                    properties:
                      token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("accepts a create whose composed schema resolves to the GET schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingWithToken"
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
components:
  schemas:
    Thing:
      type: object
      properties:
        id: {type: string}
        name: {type: string}
    ThingWithToken:
      allOf:
        - $ref: "#/components/schemas/Thing"
        - type: object
          properties:
            token: {type: string}
`), nil)
	})

	t.Run("accepts a create envelope that holds the GET schema in a property", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  thing:
                    $ref: "#/components/schemas/Thing"
                  thing_token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("accepts a create whose properties are a superset of the GET properties", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  name: {type: string}
                  token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("accepts an identical schema by the same reference", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil)
	})

	t.Run("still reports an allOf that does not include the GET schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/ThingSummary"
                  - type: object
                    properties:
                      token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things'].post.responses['201'].content['application/json']")
	})

	t.Run("still reports an envelope that does not hold the GET schema", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  summary:
                    $ref: "#/components/schemas/ThingSummary"
                  thing_token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things'].post.responses['201'].content['application/json']")
	})

	t.Run("still reports a superset of names whose property type differs", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: integer}
                  name: {type: string}
                  token: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Thing"
`+resourceResponseConsistencySchemas), nil,
			"$.paths['/things'].post.responses['201'].content['application/json']")
	})

	t.Run("does not expect a writeOnly GET property in a superset", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  name: {type: string}
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id: {type: string}
                  name: {type: string}
                  token: {type: string, writeOnly: true}
`), nil)
	})
}
