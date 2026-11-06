package rules

import (
	"iter"
	"regexp/syntax"
	"slices"
	"unicode"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// patternLengthSatisfiableUnbounded stands for an unbounded maximum length.
const patternLengthSatisfiableUnbounded = -1

// patternLengthSatisfiableExtraLevels is how many lengths beyond minLength are
// searched before the pattern is assumed to be satisfiable.
const patternLengthSatisfiableExtraLevels = 4096

// patternLengthSatisfiableMaxKernels bounds the automaton states per length.
const patternLengthSatisfiableMaxKernels = 2000

// PatternLengthSatisfiable returns the extra-pattern-length-satisfiable rule.
//
// # What it checks
//
// A string schema that has a pattern together with minLength or maxLength must
// be satisfiable: at least one string of an allowed length must be able to
// match the pattern. The rule computes the shortest and longest string the
// pattern can match, in characters, and compares that range to the length
// constraints. A pattern that can only match strings shorter than minLength or
// longer than maxLength, or that no string of an allowed length can satisfy,
// is reported. Anchors are taken into account: a pattern
// anchored at both ends matches only strings within its own length range, while
// an unanchored pattern also matches longer strings that merely contain a match.
//
// # Why
//
// A pattern and length limits that exclude each other make the schema reject
// every value. The mistake is easy to make when a limit is changed without the
// pattern, and it is only noticed when clients can no longer send valid data.
//
// # Where it comes from
//
// Specmatic (generation of values from a pattern that satisfy minLength and
// maxLength).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Code:
//	      type: string
//	      pattern: "^[A-Z]{3}$"
//	      minLength: 5
//
// # Good
//
//	components:
//	  schemas:
//	    Code:
//	      type: string
//	      pattern: "^[A-Z]{3}$"
//	      minLength: 3
//	      maxLength: 3
//
// # Known false positives / limitations
//
// The pattern is compiled to an RE2 automaton and searched exhaustively, length
// by length, over the character classes of the pattern (one representative
// character per distinct class boundary), so contradictory classes, anchors and
// repetition counts are all found, not only the length range. Word boundaries
// are assumed to be satisfiable. The pattern is parsed as an RE2 expression;
// patterns that use ECMA 262 features that RE2 lacks (look-ahead, look-behind,
// back-references) cannot be parsed and are skipped, and so are invalid
// patterns. When the search would need more than a few thousand lengths or
// automaton states the pattern is assumed to be satisfiable. Lengths are counted
// in characters (code points), not UTF-16 units. Multi-line anchors are treated
// like anchors of the whole text. A schema whose minLength exceeds its
// maxLength is left to other rules.
func PatternLengthSatisfiable() Rule {
	return New(Spec{
		Name:        "pattern-length-satisfiable",
		Description: "A pattern must be able to match a string within the schema minLength and maxLength.",
		HowToFix:    "Change the pattern or the minLength and maxLength so that a string can satisfy both.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkPatternLengthSatisfiable),
	})
}

func checkPatternLengthSatisfiable(_ *Call, schema oas.Schema) []Finding {
	key, pattern := oas.Entry(schema.Node, "pattern")
	if !oas.IsScalar(pattern) {
		return nil
	}

	minimum, _ := oas.Int(oas.Get(schema.Node, "minLength"))
	declared, bounded := oas.Int(oas.Get(schema.Node, "maxLength"))
	maximum := patternLengthSatisfiableMaximum(declared, bounded)

	return When(
		(maximum == patternLengthSatisfiableUnbounded || minimum <= maximum) &&
			!patternLengthSatisfiableOK(pattern.Value, minimum, maximum),
		func() Finding {
			return Finding{
				Node:    pattern,
				Path:    oas.Join(schema.Path, key.Value),
				Message: "pattern '" + pattern.Value + "' cannot match a string within minLength and maxLength",
			}
		},
	)
}

// patternLengthSatisfiableMaximum returns the declared maximum, or the unbounded
// marker when none is declared.
func patternLengthSatisfiableMaximum(declared int, bounded bool) int {
	if !bounded {
		return patternLengthSatisfiableUnbounded
	}

	return declared
}

// patternLengthSatisfiableLimit returns the deepest length worth exploring.
func patternLengthSatisfiableLimit(minimum, maximum int) int {
	limit := minimum + patternLengthSatisfiableExtraLevels
	if maximum == patternLengthSatisfiableUnbounded {
		return limit
	}

	return min(limit, maximum)
}

// patternLengthSatisfiableMachine is a compiled pattern with the representative
// characters of its distinct character classes.
type patternLengthSatisfiableMachine struct {
	prog    *syntax.Prog
	classes []rune
}

// patternLengthSatisfiableOK reports whether some string whose length lies in
// [minimum, maximum] can match the pattern. A negative maximum is unbounded.
// Patterns that cannot be parsed or compiled are assumed to be satisfiable.
func patternLengthSatisfiableOK(pattern string, minimum, maximum int) bool {
	tree, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return true
	}

	prog, err := syntax.Compile(tree.Simplify())
	if err != nil {
		return true
	}

	machine := &patternLengthSatisfiableMachine{prog: prog, classes: patternLengthSatisfiableRepresentatives(prog)}

	return machine.search(minimum, maximum, patternLengthSatisfiableLimit(minimum, maximum))
}

// patternLengthSatisfiableRepresentatives returns one character for every
// interval of characters that the automaton cannot tell apart.
func patternLengthSatisfiableRepresentatives(prog *syntax.Prog) []rune {
	points := slices.Collect(func(yield func(rune) bool) {
		if !yield(0) {
			return
		}

		for index := range prog.Inst {
			for point := range patternLengthSatisfiablePoints(&prog.Inst[index]) {
				if !yield(point) {
					return
				}
			}
		}
	})

	return slices.Compact(slices.Sorted(
		slices.Values(slices.DeleteFunc(points, func(point rune) bool { return point > unicode.MaxRune })),
	))
}

// patternLengthSatisfiableStarts yields the start index of each range pair.
func patternLengthSatisfiableStarts(length int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for index := range length {
			if index%2 == 0 && !yield(index) {
				return
			}
		}
	}
}

// patternLengthSatisfiablePoints yields the interval boundaries of an
// instruction.
func patternLengthSatisfiablePoints(inst *syntax.Inst) iter.Seq[rune] {
	return func(yield func(rune) bool) {
		if inst.Op == syntax.InstRuneAnyNotNL && (!yield('\n') || !yield('\n'+1)) {
			return
		}

		if inst.Op != syntax.InstRune && inst.Op != syntax.InstRune1 {
			return
		}

		for index := range slices.Collect(patternLengthSatisfiableStarts(len(inst.Rune))) {
			for point := range patternLengthSatisfiableRangePoints(inst, index) {
				if !yield(point) {
					return
				}
			}
		}
	}
}

// patternLengthSatisfiableRangePoints yields the boundaries of one range of a
// rune instruction.
func patternLengthSatisfiableRangePoints(inst *syntax.Inst, index int) iter.Seq[rune] {
	return func(yield func(rune) bool) {
		low, high := inst.Rune[index], inst.Rune[min(index+1, len(inst.Rune)-1)]

		if !yield(low) || !yield(high+1) {
			return
		}

		for point := range patternLengthSatisfiableFolds(
			low,
			high,
			syntax.Flags(inst.Arg), //nolint:gosec // regexp/syntax stores flags in a uint32 that fits
		) {
			if !yield(point) {
				return
			}
		}
	}
}

// patternLengthSatisfiableFoldRange bounds the characters examined for folds.
const patternLengthSatisfiableFoldRange = 256

// patternLengthSatisfiableFolds yields the case variants of a small range as
// interval boundaries, when the instruction folds case.
func patternLengthSatisfiableFolds(low, high rune, flags syntax.Flags) iter.Seq[rune] {
	return func(yield func(rune) bool) {
		if flags&syntax.FoldCase == 0 || high-low > patternLengthSatisfiableFoldRange {
			return
		}

		for offset := range high - low + 1 {
			char := low + offset
			if !patternLengthSatisfiableCycle(char, unicode.SimpleFold(char), yield) {
				return
			}
		}
	}
}

// patternLengthSatisfiableCycle yields the fold boundaries of the orbit that
// starts after char, and reports whether the consumer wants more.
func patternLengthSatisfiableCycle(char, fold rune, yield func(rune) bool) bool {
	return fold == char ||
		(yield(fold) && yield(fold+1) && patternLengthSatisfiableCycle(char, unicode.SimpleFold(fold), yield))
}

// matches reports whether the rune instruction accepts the character.
func (m *patternLengthSatisfiableMachine) matches(inst *syntax.Inst, char rune) bool {
	if inst.Op == syntax.InstRuneAny {
		return true
	}

	if inst.Op == syntax.InstRuneAnyNotNL {
		return char != '\n'
	}

	return inst.MatchRune(char)
}

// emptyOK reports whether an empty width assertion holds at the position.
func (m *patternLengthSatisfiableMachine) emptyOK(op syntax.EmptyOp, atStart, atEnd bool) bool {
	if op&(syntax.EmptyBeginText|syntax.EmptyBeginLine) != 0 && !atStart {
		return false
	}

	return op&(syntax.EmptyEndText|syntax.EmptyEndLine) == 0 || atEnd
}

// successors returns the instructions that follow the instruction without
// consuming a character.
func (m *patternLengthSatisfiableMachine) successors(pc int, atStart, atEnd bool) []int {
	inst := &m.prog.Inst[pc]

	switch inst.Op {
	case syntax.InstAlt, syntax.InstAltMatch:
		return []int{int(inst.Out), int(inst.Arg)}
	case syntax.InstCapture, syntax.InstNop:
		return []int{int(inst.Out)}
	case syntax.InstEmptyWidth:
		if m.emptyOK(
			syntax.EmptyOp(inst.Arg), //nolint:gosec // regexp/syntax stores the op in a uint32 that fits
			atStart,
			atEnd,
		) {
			return []int{int(inst.Out)}
		}
	case syntax.InstMatch, syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny,
		syntax.InstRuneAnyNotNL, syntax.InstFail:
	}

	return nil
}

// reach returns the sorted set of instructions reachable from the frontier
// without consuming a character, given the sorted set already seen.
func (m *patternLengthSatisfiableMachine) reach(seen, frontier []int, atStart, atEnd bool) []int {
	if len(frontier) == 0 {
		return seen
	}

	all := slices.Sorted(slices.Values(slices.Concat(seen, frontier)))

	next := slices.Compact(slices.Sorted(func(yield func(int) bool) {
		for _, pc := range frontier {
			for _, target := range m.successors(pc, atStart, atEnd) {
				if _, found := slices.BinarySearch(all, target); !found && !yield(target) {
					return
				}
			}
		}
	}))

	return m.reach(all, next, atStart, atEnd)
}

// closure follows the instructions that consume no character and returns the
// character instructions reached and whether a match was reached.
func (m *patternLengthSatisfiableMachine) closure(kernel []int, atStart, atEnd bool) ([]int, bool) {
	reached := m.reach(nil, slices.Compact(slices.Sorted(slices.Values(kernel))), atStart, atEnd)

	matched := slices.ContainsFunc(reached, func(pc int) bool { return m.prog.Inst[pc].Op == syntax.InstMatch })

	runes := slices.DeleteFunc(slices.Clone(reached), func(pc int) bool {
		switch m.prog.Inst[pc].Op {
		case syntax.InstRune, syntax.InstRune1, syntax.InstRuneAny, syntax.InstRuneAnyNotNL:
			return false
		case syntax.InstAlt, syntax.InstAltMatch, syntax.InstCapture, syntax.InstEmptyWidth,
			syntax.InstFail, syntax.InstMatch, syntax.InstNop:
		}

		return true
	})

	return runes, matched
}

// step returns the automaton states reached from the character instructions
// by consuming the character.
func (m *patternLengthSatisfiableMachine) step(runes []int, char rune) []int {
	return slices.Compact(slices.Sorted(func(yield func(int) bool) {
		for _, pc := range runes {
			inst := &m.prog.Inst[pc]
			if m.matches(inst, char) && !yield(int(inst.Out)) {
				return
			}
		}
	}))
}

// patternLengthSatisfiableLevel is what one automaton state set offers at one
// length: the character instructions it can consume and whether it accepts.
type patternLengthSatisfiableLevel struct {
	runes    []int
	accepted bool
}

// analyse closes the state set at the length.
func (m *patternLengthSatisfiableMachine) analyse(kernel []int, length, minimum int) patternLengthSatisfiableLevel {
	start := slices.Concat(kernel, []int{m.prog.Start})

	runes, matched := m.closure(start, length == 0, false)
	_, atEnd := m.closure(start, length == 0, true)

	return patternLengthSatisfiableLevel{runes: runes, accepted: matched || (atEnd && length >= minimum)}
}

// search walks the lengths from zero to the limit and reports whether some
// length in [minimum, maximum] has an accepting string. It assumes the pattern
// is satisfiable when the limit is reached or the states grow too large.
func (m *patternLengthSatisfiableMachine) search(minimum, maximum, limit int) bool {
	return m.searchFrom([][]int{nil}, 0, minimum, maximum, limit)
}

// searchFrom continues the search at the length with the distinct state sets.
func (m *patternLengthSatisfiableMachine) searchFrom(level [][]int, length, minimum, maximum, limit int) bool {
	if length > limit {
		return maximum != patternLengthSatisfiableUnbounded && limit < maximum
	}

	levels := slices.Collect(func(yield func(patternLengthSatisfiableLevel) bool) {
		for _, kernel := range level {
			if !yield(m.analyse(kernel, length, minimum)) {
				return
			}
		}
	})

	if slices.ContainsFunc(levels, func(item patternLengthSatisfiableLevel) bool { return item.accepted }) {
		return true
	}

	if !slices.ContainsFunc(levels, func(item patternLengthSatisfiableLevel) bool { return len(item.runes) > 0 }) {
		return false
	}

	next := slices.CompactFunc(
		slices.SortedFunc(func(yield func([]int) bool) {
			for _, item := range levels {
				for _, char := range m.classes {
					if !yield(m.step(item.runes, char)) {
						return
					}
				}
			}
		}, slices.Compare[[]int]),
		slices.Equal[[]int],
	)

	if len(next) > patternLengthSatisfiableMaxKernels {
		return true
	}

	return m.searchFrom(next, length+1, minimum, maximum, limit)
}
