package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestCollectionResponseHasNamedArray(t *testing.T) {
	t.Parallel()

	rule := rules.CollectionResponseHasNamedArray()

	t.Run("flags a list response with a differently named array", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      operationId: list_things
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  results:
                    type: array
                    items:
                      type: string
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.paths['/things'].get.responses['200'].content['application/json'].schema", results[0].Path)
		require.Contains(t, results[0].Message, "things")
	})

	t.Run("accepts the correctly named array", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      operationId: listThings
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ThingCollection"
components:
  schemas:
    ThingCollection:
      type: object
      properties:
        things:
          type: array
          items:
            type: string
`), nil)
	})

	t.Run("detects a list through a sibling item path", func(t *testing.T) {
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
                type: object
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
`), nil, "$.paths['/things'].get.responses['200'].content['application/json'].schema")
	})

	t.Run("accepts the array inside an allOf member", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      operationId: list_things
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                allOf:
                  - type: object
                    properties:
                      things:
                        type: array
                        items:
                          type: string
`), nil)
	})

	t.Run("ignores non list operations and binary downloads", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things/{id}:
    get:
      operationId: get_thing
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
  /files:
    get:
      operationId: list_files
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: string
                format: binary
`), nil)
	})
}
