// Package seq holds small, pure helpers over iterators and slices: mapping,
// filtering, flattening, grouping and de-duplicating without the caller
// holding any mutable state.
//
// The bodies of these helpers hide the accumulation loops so that no caller
// needs one.
package seq
