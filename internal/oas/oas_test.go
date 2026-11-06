package oas_test

import (
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const document = `
openapi: 3.1.0
paths:
  /pets/{id}:
    parameters:
      - $ref: "#/components/parameters/Id"
    get:
      responses:
        "200": {description: ok}
components:
  parameters:
    Id: {name: id, in: path, required: true, schema: {type: string}}
`

func parse(t *testing.T) *yaml.Node {
	t.Helper()

	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(document), &node))

	return oas.Unwrap(&node)
}

func TestOperationsAndParameters(t *testing.T) {
	t.Parallel()

	root := parse(t)
	var found []oas.Operation
	for operation := range oas.Operations(root) {
		found = append(found, operation)
	}

	require.Len(t, found, 1)
	require.Equal(t, "$.paths['/pets/{id}'].get", found[0].Path)

	parameters := oas.Parameters(root, found[0])
	require.Len(t, parameters, 1)
	require.Equal(t, "id", parameters[0].Name())
	require.True(t, parameters[0].Inherited)
	require.True(t, oas.IsOAS31(root))
}

func TestJoin(t *testing.T) {
	t.Parallel()

	require.Equal(t, "$.a", oas.Join("$", "a"))
	require.Equal(t, "$['/x']", oas.Join("$", "/x"))
	require.Equal(t, "$.a[3]", oas.Index("$.a", 3))
}

func TestPointerAndResolve(t *testing.T) {
	t.Parallel()

	root := parse(t)
	require.NotNil(t, oas.Pointer(root, "#/components/parameters/Id"))
	require.Nil(t, oas.Pointer(root, "#/components/missing"))
	require.Nil(t, oas.Pointer(root, "other.yaml#/x"))
}

func TestUnderNegation(t *testing.T) {
	t.Parallel()

	require.True(t, oas.UnderNegation("$.components.schemas.A.not"))
	require.True(t, oas.UnderNegation("$.components.schemas.A.not.anyOf[0]"))
	require.True(t, oas.UnderNegation("$.components.schemas.A.properties.b.if.properties.c"))
	require.True(t, oas.UnderNegation("$.components.schemas.A.then.properties.x"))
	require.False(t, oas.UnderNegation("$.components.schemas.A.properties.not"))
	require.False(t, oas.UnderNegation("$.components.schemas.A.properties['else'].type"))
	require.False(t, oas.UnderNegation("$.components.schemas.Nothing.properties.x"))
}

func TestNodeAt(t *testing.T) {
	t.Parallel()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("a:\n  b-c: [x, {d: 1}]\n  \"it's\": y\n"), &root))

	require.Equal(t, "x", oas.NodeAt(&root, "$.a['b-c'][0]").Value)
	require.Equal(t, "1", oas.NodeAt(&root, oas.JoinAll("$", "a", "b-c")+"[1].d").Value)
	require.Equal(t, "y", oas.NodeAt(&root, oas.JoinAll("$", "a", "it's")).Value)
	require.Nil(t, oas.NodeAt(&root, "$.a.missing"))
	require.Nil(t, oas.NodeAt(&root, "$.a['b-c'][5]"))
	require.NotNil(t, oas.NodeAt(&root, "$"))
}
