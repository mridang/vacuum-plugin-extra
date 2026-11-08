package formats_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/formats"
)

func TestNonEmpty(t *testing.T) {
	t.Parallel()

	t.Run("returns a fresh copy", func(t *testing.T) {
		t.Parallel()

		first := formats.NonEmpty()
		first[0] = "changed"

		require.NotEqual(t, "changed", formats.NonEmpty()[0])
	})

	t.Run("lists the self-describing formats", func(t *testing.T) {
		t.Parallel()

		require.Subset(t, formats.NonEmpty(), []string{"date-time", "uri", "email", "uuid", "ipv6"})
	})
}

func TestContains(t *testing.T) {
	t.Parallel()

	list := []string{"date-time", "uri"}

	require.True(t, formats.Contains(list, "uri"))
	require.False(t, formats.Contains(list, "email"))
	require.False(t, formats.Contains(list, ""))
	require.False(t, formats.Contains([]string{""}, ""))
}
