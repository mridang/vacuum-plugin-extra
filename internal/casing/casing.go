package casing

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Casing is a naming convention.
type Casing string

const (
	// Kebab is lower case words joined by hyphens: pet-owner.
	Kebab Casing = "kebab-case"
	// Snake is lower case words joined by underscores: pet_owner.
	Snake Casing = "snake_case"
	// Camel is lowerCamelCase: petOwner.
	Camel Casing = "camelCase"
	// Pascal is UpperCamelCase: PetOwner.
	Pascal Casing = "PascalCase"
	// ScreamingSnake is upper case words joined by underscores: PET_OWNER.
	ScreamingSnake Casing = "SCREAMING_SNAKE_CASE"
	// ScreamingKebab is upper case words joined by hyphens: PET-OWNER.
	ScreamingKebab Casing = "SCREAMING-KEBAB-CASE"
	// Dot is lower case words joined by dots: pet.owner.
	Dot Casing = "dot.case"
	// Lower is lower case letters and digits only, with no separator: petowner.
	Lower Casing = "lowercase"
)

// UnknownError reports a value that names no casing.
type UnknownError struct {
	// Value is the rejected value.
	Value string
}

// Error implements error and names every allowed value.
func (e UnknownError) Error() string {
	return fmt.Sprintf("unknown casing %q: allowed values are %s", e.Value, strings.Join(Names(), ", "))
}

// All returns every supported casing.
func All() []Casing {
	return []Casing{Kebab, Snake, Camel, Pascal, ScreamingSnake, ScreamingKebab, Dot, Lower}
}

// Names returns the accepted option values, one per casing.
func Names() []string {
	return seq.Map(slices.Values(All()), func(c Casing) string {
		return string(c)
	})
}

// Parse converts an option value into a Casing. An unknown value is an
// UnknownError rather than a silent fallback.
func Parse(value string) (Casing, error) {
	if !slices.Contains(Names(), value) {
		return "", UnknownError{Value: value}
	}

	return Casing(value), nil
}

// Matches reports whether name follows the convention. Only ASCII letters and
// digits are word characters, so any other character makes the name invalid.
func (c Casing) Matches(name string) bool {
	switch c {
	case Kebab:
		return joined(name, '-', isLowerOrDigit)
	case Snake:
		return joined(name, '_', isLowerOrDigit)
	case ScreamingKebab:
		return joined(name, '-', isUpperOrDigit)
	case ScreamingSnake:
		return joined(name, '_', isUpperOrDigit)
	case Dot:
		return joined(name, '.', isLowerOrDigit)
	case Lower:
		return name != "" && !strings.ContainsFunc(name, func(r rune) bool { return !isLowerOrDigit(r) })
	case Camel:
		return camelShaped(name, isLowerOrDigit)
	case Pascal:
		return camelShaped(name, isUpper)
	default:
		return false
	}
}

// Convert rewrites name in the convention. Words are found at separators
// (hyphen, underscore, dot, space) and at lower-to-upper case changes.
// Acronyms such as ID are kept intact in the camel conventions.
func (c Casing) Convert(name string) string {
	words := Words(name)
	if len(words) == 0 {
		return name
	}

	switch c {
	case Kebab:
		return strings.ToLower(strings.Join(words, "-"))
	case Snake:
		return strings.ToLower(strings.Join(words, "_"))
	case ScreamingKebab:
		return strings.ToUpper(strings.Join(words, "-"))
	case ScreamingSnake:
		return strings.ToUpper(strings.Join(words, "_"))
	case Dot:
		return strings.ToLower(strings.Join(words, "."))
	case Lower:
		return strings.ToLower(strings.Join(words, ""))
	case Camel:
		return camelJoin(words, false, mixedCase(name))
	case Pascal:
		return camelJoin(words, true, mixedCase(name))
	default:
		return name
	}
}

// Set is the list of conventions a name may follow; a name is valid when it
// follows any one of them.
type Set []Casing

// ParseSet converts option values into a Set, failing on the first unknown one.
func ParseSet(values []string) (Set, error) {
	set := make(Set, 0, len(values))

	for _, value := range values {
		parsed, err := Parse(value)
		if err != nil {
			return nil, err
		}

		set = append(set, parsed)
	}

	return set, nil
}

// Matches reports whether name follows at least one convention of the set. An
// empty set accepts every name.
func (s Set) Matches(name string) bool {
	return len(s) == 0 || slices.ContainsFunc(s, func(c Casing) bool {
		return c.Matches(name)
	})
}

// Convert rewrites name in the first convention of the set.
func (s Set) Convert(name string) string {
	if len(s) == 0 {
		return name
	}

	return s[0].Convert(name)
}

// String lists the conventions for a message: "snake_case or PascalCase".
func (s Set) String() string {
	return strings.Join(seq.Map(slices.Values(s), func(c Casing) string {
		return string(c)
	}), " or ")
}

// Words splits name into words at separators and at case changes.
func Words(name string) []string {
	return seq.FlatMapSlice(
		strings.FieldsFunc(name, func(r rune) bool {
			return r == '-' || r == '_' || r == '.' || r == ' '
		}),
		splitCamel,
	)
}

// splitCamel splits one separator-free segment where a new capital starts.
func splitCamel(segment string) []string {
	runes := []rune(segment)
	starts := slices.Concat(
		[]int{0},
		seq.FlatMapIndexed(runes, func(i int, _ rune) []int {
			if !boundary(runes, i) {
				return nil
			}

			return []int{i}
		}),
		[]int{len(runes)},
	)

	return seq.FlatMapIndexed(starts[:len(starts)-1], func(n int, start int) []string {
		return []string{string(runes[start:starts[n+1]])}
	})
}

// boundary reports whether a new word starts at index i.
func boundary(runes []rune, i int) bool {
	if i == 0 || !unicode.IsUpper(runes[i]) {
		return false
	}

	previous := runes[i-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}

	return unicode.IsUpper(previous) && i+1 < len(runes) && unicode.IsLower(runes[i+1])
}

// camelJoin joins words as camel case, capitalising the first word when upper
// is set. With acronyms, all-capital words are kept as they are except in the
// first word of lowerCamelCase.
func camelJoin(words []string, upper bool, acronyms bool) string {
	return strings.Join(seq.FlatMapIndexed(words, func(i int, word string) []string {
		return []string{camelWord(word, upper || i > 0, acronyms)}
	}), "")
}

// mixedCase reports whether name has a lower-case letter, which is what tells
// an acronym such as ID from a name written entirely in capitals.
func mixedCase(name string) bool {
	return strings.ContainsFunc(name, unicode.IsLower)
}

// camelWord capitalises or lowers one word.
func camelWord(word string, capitalise bool, acronyms bool) string {
	if acronyms && word == strings.ToUpper(word) {
		if capitalise {
			return word
		}

		return strings.ToLower(word)
	}

	lower := strings.ToLower(word)
	if !capitalise {
		return lower
	}

	first, size := utf8.DecodeRuneInString(lower)

	return string(unicode.ToUpper(first)) + lower[size:]
}

// joined reports whether name is one or more non-empty words of word
// characters separated by single separators.
func joined(name string, separator rune, word func(rune) bool) bool {
	return name != "" && !slices.ContainsFunc(strings.Split(name, string(separator)), func(part string) bool {
		return part == "" || strings.ContainsFunc(part, func(r rune) bool { return !word(r) })
	})
}

// camelShaped reports whether name starts with a character accepted by first
// and continues with ASCII letters and digits.
func camelShaped(name string, first func(rune) bool) bool {
	runes := []rune(name)

	return len(runes) > 0 && first(runes[0]) && !slices.ContainsFunc(runes, func(r rune) bool {
		return !isUpper(r) && !isLowerOrDigit(r)
	})
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isLowerOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || isDigit(r)
}

func isUpperOrDigit(r rune) bool {
	return isUpper(r) || isDigit(r)
}
