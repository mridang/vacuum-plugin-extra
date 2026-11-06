package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestResponseStatusCodeSemantics(t *testing.T) {
	t.Parallel()

	rule := rules.ResponseStatusCodeSemantics()

	t.Run("flags 422 and 302", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
        "302":
          description: moved
          content:
            text/plain: {}
        "422":
          description: invalid
`), nil)

		require.ElementsMatch(t, []string{
			"$.paths['/things'].get.responses['302']",
			"$.paths['/things'].get.responses['422']",
		}, ruletest.Paths(results))
	})

	t.Run("flags a missing success code", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "404":
          description: missing
`), nil, "$.paths['/things'].get.responses")
	})

	t.Run("flags 101 with a success code and 202 with another 2xx", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "101":
          description: switching
        "200":
          description: ok
    post:
      responses:
        "200":
          description: ok
        "202":
          description: accepted
`), nil, "$.paths['/things'].get.responses['101']", "$.paths['/things'].post.responses")
	})

	t.Run("flags a 204 with content and redirects without content", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    delete:
      responses:
        "204":
          description: gone
          content:
            application/json: {}
        "301":
          description: moved
`), nil,
			"$.paths['/things'].delete.responses['204'].content",
			"$.paths['/things'].delete.responses")
	})

	t.Run("flags creates, PUT and PATCH with the wrong code", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      operationId: createThing
      responses:
        "200":
          description: ok
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json: {}
    put:
      responses:
        "204":
          description: updated
    patch:
      responses:
        "201":
          description: created
`), nil,
			"$.paths['/things'].post.responses",
			"$.paths['/things/{id}'].put.responses",
			"$.paths['/things/{id}'].patch.responses")
	})

	t.Run("accepts well formed operations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /things:
    post:
      operationId: createThing
      responses:
        "201":
          description: created
        "400":
          description: invalid
  /things/{id}:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json: {}
    put:
      responses:
        "200":
          description: ok
    patch:
      responses:
        "202":
          description: accepted
    delete:
      responses:
        "204":
          description: gone
`), nil)
	})

	t.Run("allows 204 for a create when the resource has no body", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      responses:
        "204":
          description: created
  /things/{id}:
    get:
      responses:
        "204":
          description: no body
`), nil)
	})

	t.Run("uses the IBM messages and reports redirects without content", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    get:
      responses:
        "200":
          description: ok
        "301":
          description: moved
        "307":
          description: moved
        "422":
          description: invalid
`), nil)

		require.Equal(t, []string{
			"$.paths['/things'].get.responses['422']",
			"$.paths['/things'].get.responses",
		}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results),
			"Operation responses should use status code 400 instead of 422 for invalid request payloads")
		require.Contains(t, ruletest.Messages(results),
			"A 301, 302, 305 or 307 response should include a response body, "+
				"use a different status code for responses without content")
	})

	t.Run("reports a create without any success code twice", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      operationId: Create_thing
      responses:
        "400":
          description: bad
`), nil)

		require.Equal(t, []string{"$.paths['/things'].post.responses", "$.paths['/things'].post.responses"},
			ruletest.Paths(results))
		require.ElementsMatch(t, []string{
			"Operation responses should include at least one success status code (2xx)",
			"A 201 or 202 status code should be returned by a 'create' operation",
		}, ruletest.Messages(results))
	})

	t.Run("allows PUT 204 only when the same path GET returns 204", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /minimal/{id}:
    get:
      responses:
        "204":
          description: none
    put:
      responses:
        "204":
          description: updated
  /full/{id}:
    get:
      responses:
        "200":
          description: ok
    put:
      responses:
        "204":
          description: updated
`), nil, "$.paths['/full/{id}'].put.responses")
	})

	t.Run("ignores operations that only define range or default responses", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /things:
    post:
      operationId: createThing
      responses:
        default:
          description: whatever
`), nil)
	})

	t.Run("create without a sibling path may return 204", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /jobs:
    post:
      operationId: createJob
      responses:
        "204":
          description: done
`), nil)
	})
}
