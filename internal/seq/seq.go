package seq

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
)

// FlatMap returns the concatenation of f applied to every item.
func FlatMap[T, R any](items iter.Seq[T], f func(T) []R) []R {
	return slices.Collect(func(yield func(R) bool) {
		for item := range items {
			for _, result := range f(item) {
				if !yield(result) {
					return
				}
			}
		}
	})
}

// FlatMapSlice is FlatMap over a slice.
func FlatMapSlice[T, R any](items []T, f func(T) []R) []R {
	return FlatMap(slices.Values(items), f)
}

// FlatMapIndexed is FlatMap over a slice, passing the index of each item.
func FlatMapIndexed[T, R any](items []T, f func(int, T) []R) []R {
	return slices.Collect(func(yield func(R) bool) {
		for i, item := range items {
			for _, result := range f(i, item) {
				if !yield(result) {
					return
				}
			}
		}
	})
}

// FlatMapPairs is FlatMap over the pairs of an iterator.
func FlatMapPairs[K, V, R any](pairs iter.Seq2[K, V], f func(K, V) []R) []R {
	return slices.Collect(func(yield func(R) bool) {
		for key, value := range pairs {
			for _, result := range f(key, value) {
				if !yield(result) {
					return
				}
			}
		}
	})
}

// Map returns f applied to every item.
func Map[T, R any](items iter.Seq[T], f func(T) R) []R {
	return slices.Collect(func(yield func(R) bool) {
		for item := range items {
			if !yield(f(item)) {
				return
			}
		}
	})
}

// Filter returns the items for which keep is true.
func Filter[T any](items iter.Seq[T], keep func(T) bool) []T {
	return slices.Collect(func(yield func(T) bool) {
		for item := range items {
			if keep(item) && !yield(item) {
				return
			}
		}
	})
}

// GroupBy returns the items grouped by key, each group in input order.
//
// The items are stably sorted by a textual form of their key and then cut into
// runs of equal keys, so the grouping needs no mutable state.
func GroupBy[T any, K comparable](items iter.Seq[T], key func(T) K) map[K][]T {
	keyed := Map(items, func(item T) groupEntry[T, K] {
		return groupEntry[T, K]{key: key(item), item: item}
	})
	slices.SortStableFunc(keyed, func(a, b groupEntry[T, K]) int {
		return cmp.Compare(orderOf(a.key), orderOf(b.key))
	})

	return maps.Collect(func(yield func(K, []T) bool) {
		for i, entry := range keyed {
			if i > 0 && keyed[i-1].key == entry.key {
				continue
			}

			if !yield(entry.key, Map(slices.Values(runFrom(keyed[i:])), func(e groupEntry[T, K]) T { return e.item })) {
				return
			}
		}
	})
}

type groupEntry[T any, K comparable] struct {
	key  K
	item T
}

func orderOf[K comparable](key K) string {
	if text, ok := any(key).(string); ok {
		return text
	}

	return fmt.Sprint(key)
}

// runFrom returns the leading entries of sorted that share the key of the first.
func runFrom[T any, K comparable](sorted []groupEntry[T, K]) []groupEntry[T, K] {
	end := slices.IndexFunc(sorted, func(e groupEntry[T, K]) bool { return e.key != sorted[0].key })

	if end < 0 {
		return sorted
	}

	return sorted[:end]
}

// Duplicates returns the keys that occur more than once, sorted.
func Duplicates[T any, K cmp.Ordered](items iter.Seq[T], key func(T) K) []K {
	groups := GroupBy(items, key)

	return slices.Sorted(maps.Keys(maps.Collect(Pairs(groups, func(_ K, group []T) bool {
		return len(group) > 1
	}))))
}

// Pairs yields the entries of a map for which keep is true.
func Pairs[K comparable, V any](entries map[K]V, keep func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for key, value := range entries {
			if keep(key, value) && !yield(key, value) {
				return
			}
		}
	}
}

// Occurrences returns, for every key, the items that carry it, in input order.
func Occurrences[T any, K comparable](items []T, key func(T) K) map[K][]T {
	return GroupBy(slices.Values(items), key)
}
