package oas_test

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const markedDocument = `
components:
  schemas:
    Stamp:
      type: string
      readOnly: true
    Wrapped:
      allOf:
        - $ref: "#/components/schemas/Stamp"
    Plain:
      type: string
    Object:
      type: object
      required: [id, name]
      properties:
        id: {type: string, readOnly: true}
        name: {type: string}
`

func parseMarked(t *testing.T) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(markedDocument), &node))

	return oas.Unwrap(&node)
}

func TestMarked(t *testing.T) {
	t.Parallel()

	root := parseMarked(t)
	schemas := oas.Get(oas.Get(root, "components"), "schemas")

	t.Run("reads the keyword on the schema itself", func(t *testing.T) {
		t.Parallel()

		require.True(t, oas.Marked(root, oas.Get(schemas, "Stamp"), "readOnly"))
	})

	t.Run("follows a $ref and an allOf member", func(t *testing.T) {
		t.Parallel()

		require.True(t, oas.Marked(root, &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "#/components/schemas/Wrapped"},
		}}, "readOnly"))
	})

	t.Run("is false when the keyword is absent, false or not a schema", func(t *testing.T) {
		t.Parallel()

		require.False(t, oas.Marked(root, oas.Get(schemas, "Plain"), "readOnly"))
		require.False(t, oas.Marked(root, oas.Get(schemas, "Stamp"), "writeOnly"))
		require.False(t, oas.Marked(root, nil, "readOnly"))
	})
}

func TestWithout(t *testing.T) {
	t.Parallel()

	root := parseMarked(t)
	object := oas.Get(oas.Get(oas.Get(root, "components"), "schemas"), "Object")

	t.Run("removes a dropped property and its required name", func(t *testing.T) {
		t.Parallel()

		trimmed := oas.Without(object, func(property *yaml.Node) bool {
			return oas.Marked(root, property, "readOnly")
		})

		require.Equal(t, []string{"name"}, oas.Keys(oas.Get(trimmed, "properties")))
		require.Equal(t, []string{"name"}, oas.Strings(oas.Get(trimmed, "required")))
		require.Equal(t, []string{"id", "name"}, oas.Strings(oas.Get(object, "required")))
		require.Equal(t, []string{"id", "name"}, oas.Keys(oas.Get(object, "properties")))
	})

	t.Run("returns the schema itself when nothing is dropped", func(t *testing.T) {
		t.Parallel()

		require.Same(t, object, oas.Without(object, func(*yaml.Node) bool { return false }))
	})
}
