package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestOperationIDVerbMatchesMethod(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDVerbMatchesMethod()

	t.Run("flags verbs and nouns that do not match", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /bands:
    get:
      operationId: get_bands
      responses:
        "200":
          description: ok
    post:
      operationId: add_band
      responses:
        "200":
          description: ok
  /bands/{id}:
    get:
      operationId: list_band
      responses:
        "200":
          description: ok
    patch:
      operationId: modify_band
      responses:
        "200":
          description: ok
    delete:
      operationId: delete_bands
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/bands'].get.operationId",
			"$.paths['/bands'].post.operationId",
			"$.paths['/bands/{id}'].get.operationId",
			"$.paths['/bands/{id}'].patch.operationId",
			"$.paths['/bands/{id}'].delete.operationId",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results),
			"operationIds should follow naming convention: operationId should be list_bands but it's get_bands instead")
		require.Contains(t, ruletest.Messages(results),
			"operationIds should follow naming convention: operationId should be get_band but it's list_band instead")
	})

	t.Run("accepts the conventional names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /bands:
    get:
      operationId: list_bands
      responses:
        "200":
          description: ok
    post:
      operationId: create_band
      responses:
        "200":
          description: ok
  /bands/{id}:
    get:
      operationId: get_band
      responses:
        "200":
          description: ok
    put:
      operationId: replace_band
      responses:
        "200":
          description: ok
    patch:
      operationId: update_band
      responses:
        "200":
          description: ok
    delete:
      operationId: delete_band
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("accepts nested verbs and singularised nouns", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /bands/{band}/tracks:
    put:
      operationId: set_band_tracks
      responses:
        "200":
          description: ok
    delete:
      operationId: unset_band_tracks
      responses:
        "200":
          description: ok
  /bands/{band}/albums:
    get:
      operationId: list_band_albums
      responses:
        "200":
          description: ok
  /bands/{band}/albums/{album}:
    get:
      operationId: check_band_album
      responses:
        "200":
          description: ok
    put:
      operationId: add_band_album
      responses:
        "200":
          description: ok
    delete:
      operationId: remove_band_album
      responses:
        "200":
          description: ok
  /v2/people/{id}/categories/{category}:
    get:
      operationId: get_person_category
      responses:
        "200":
          description: ok
  /things/{id}/metadata/{key}:
    delete:
      operationId: delete_thing_metadata
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("flags the wrong noun for irregular words", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /v1/people:
    get:
      operationId: list_persons
      responses:
        "200":
          description: ok
  /v1/people/{id}:
    get:
      operationId: get_people
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/v1/people'].get.operationId",
			"$.paths['/v1/people/{id}'].get.operationId",
		}, ruletest.Paths(results))
		require.Contains(
			t,
			ruletest.Messages(results),
			"operationIds should follow naming convention: operationId should be get_person but it's get_people instead",
		)
	})

	t.Run("non strict mode only checks the verb", func(t *testing.T) {
		t.Parallel()

		doc := ruletest.Doc30(`
paths:
  /bands:
    get:
      operationId: listAnything
      responses:
        "200":
          description: ok
    post:
      operationId: add_band
      responses:
        "200":
          description: ok
  /bands/{id}:
    get:
      operationId: get_whatever
      responses:
        "200":
          description: ok
`)
		results := ruletest.Run(t, rule, doc, ruletest.Options{"strict": false})

		require.ElementsMatch(t, []string{"$.paths['/bands'].post.operationId"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results),
			"operationIds should follow naming convention: operationId verb should be create")
	})

	t.Run("does not constrain non resource paths or missing ids", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /health:
    get:
      operationId: ping
      responses:
        "200":
          description: ok
    post:
      responses:
        "200":
          description: ok
`), nil)
	})
}
