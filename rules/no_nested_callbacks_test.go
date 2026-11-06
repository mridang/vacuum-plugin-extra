package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoNestedCallbacks(t *testing.T) {
	t.Parallel()

	rule := rules.NoNestedCallbacks()

	t.Run("flags callbacks inside a callback", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths:
  /subscribe:
    post:
      callbacks:
        onEvent:
          "{$request.body#/url}":
            post:
              callbacks:
                onAck:
                  "{$request.body#/ack}":
                    post:
                      responses:
                        "200":
                          description: ok
              responses:
                "200":
                  description: ok
      responses:
        "200":
          description: ok
`), nil)

		require.Equal(t, []string{
			"$.paths['/subscribe'].post.callbacks.onEvent['{$request.body#/url}'].post.callbacks",
		}, ruletest.Paths(results))
	})

	t.Run("flags nested callbacks in components", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
components:
  callbacks:
    onEvent:
      "{$request.body#/url}":
        post:
          callbacks:
            onAck:
              "{$request.body#/ack}":
                post:
                  responses:
                    "200":
                      description: ok
          responses:
            "200":
              description: ok
`), nil, "$.components.callbacks.onEvent['{$request.body#/url}'].post.callbacks")
	})

	t.Run("accepts a single level of callbacks", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /subscribe:
    post:
      callbacks:
        onEvent:
          "{$request.body#/url}":
            post:
              responses:
                "200":
                  description: ok
      responses:
        "200":
          description: ok
`), nil)
	})
}
