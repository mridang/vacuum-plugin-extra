package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestTracingHeadersRequired(t *testing.T) {
	t.Parallel()

	rule := rules.TracingHeadersRequired()

	t.Run("flags path items whose operation parameters lack tracing headers", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
    post:
      parameters:
        - name: X-B3-TraceId
          in: header
          schema:
            type: string
      responses:
        "201":
          description: created
  /owners:
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{"$.paths['/pets']"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "X-B3-Traceid")
	})

	t.Run("also checks operations without parameters when strict", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /owners:
    get:
      responses:
        "200":
          description: ok
`), ruletest.Options{"requireOnAllOperations": true}, "$.paths['/owners']")
	})

	t.Run("accepts the B3 pair on each operation in any case", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: x-b3-traceid
          in: header
          schema:
            type: string
        - name: X-B3-SpanId
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("accepts valid path item parameters whatever the operations declare", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: traceparent
        in: header
        schema:
          type: string
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("does not merge B3 headers split over path item and operation", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    parameters:
      - name: x-b3-traceid
        in: header
        schema:
          type: string
    get:
      parameters:
        - name: X-B3-SpanId
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets']")
	})

	t.Run("rejects two traceparent headers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: traceparent
          in: header
          schema:
            type: string
        - name: Traceparent
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets']")
	})

	t.Run("accepts a single traceparent header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      parameters:
        - $ref: "#/components/parameters/Trace"
      responses:
        "200":
          description: ok
components:
  parameters:
    Trace:
      name: traceparent
      in: header
      schema:
        type: string
`), nil)
	})

	t.Run("does not accept a traceparent query parameter", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: traceparent
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets']")
	})
}
