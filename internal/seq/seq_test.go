package seq_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

func TestFlatMapAndMap(t *testing.T) {
	t.Parallel()

	items := slices.Values([]int{1, 2, 3})

	require.Equal(t, []int{1, 1, 2, 2, 3, 3}, seq.FlatMap(items, func(i int) []int { return []int{i, i} }))
	require.Equal(t, []int{2, 4, 6}, seq.Map(items, func(i int) int { return i * 2 }))
	require.Equal(t, []int{2}, seq.Filter(items, func(i int) bool { return i == 2 }))
}

func TestFlatMapIndexed(t *testing.T) {
	t.Parallel()

	got := seq.FlatMapIndexed(
		[]string{"a", "b"},
		func(i int, s string) []string { return []string{s + string(rune('0'+i))} },
	)

	require.Equal(t, []string{"a0", "b1"}, got)
}

func TestDuplicates(t *testing.T) {
	t.Parallel()

	got := seq.Duplicates(slices.Values([]string{"b", "a", "b", "c", "a", "d"}), func(s string) string { return s })

	require.Equal(t, []string{"a", "b"}, got)
	require.Empty(t, seq.Duplicates(slices.Values([]string{"x"}), func(s string) string { return s }))
}

func TestGroupBy(t *testing.T) {
	t.Parallel()

	got := seq.GroupBy(slices.Values([]int{1, 2, 3, 4}), func(i int) bool { return i%2 == 0 })

	require.Equal(t, map[bool][]int{true: {2, 4}, false: {1, 3}}, got)
}
