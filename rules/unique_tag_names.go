package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// uniqueTagNamesCaseOption is the name of the case-insensitivity option.
const uniqueTagNamesCaseOption = "caseInsensitive"

// UniqueTagNames returns the extra-unique-tag-names rule.
//
// # What it checks
//
// The names in the top-level tags list must be unique. A tag name is the key
// that operations use to group themselves, so two tag objects with one name
// make the grouping ambiguous and documentation tools show one of them at
// random. Every repeated entry after the first is reported at its name; unlike
// some linters that stop at the first duplicate, all duplicates are listed.
//
// # Where it comes from
//
// Three near-identical rules merged into one: duplicate-tag-name from the
// spec-validator, tags-unique-names from Spectral and unique-tag-names from
// rate-my-openapi.
//
// # Options
//
//   - caseInsensitive: compare names ignoring case and surrounding white
//     space, so "Pets" and "pets" count as duplicates. Default: false.
//
// # Bad
//
//	tags:
//	  - name: pets
//	  - name: pets
//
// # Good
//
//	tags:
//	  - name: pets
//	  - name: owners
//
// # Known false positives
//
// None known. Tags without a name are ignored here; other rules require the
// name.
func UniqueTagNames() Rule {
	return New(Spec{
		Name:        "unique-tag-names",
		Description: "Tag names in the top-level tags list must be unique.",
		HowToFix:    "Rename or merge the duplicated tags.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        uniqueTagNamesCaseOption,
			Description: "Compare tag names ignoring case.",
			Default:     false,
		}},
		Check: OnDocument(checkUniqueTagNames),
	})
}

// uniqueTagNamesEntry is a tag with a scalar name and its comparison key.
type uniqueTagNamesEntry struct {
	index int
	name  *yaml.Node
	key   string
}

// checkUniqueTagNames reports every tag whose name equals that of an earlier
// tag.
func checkUniqueTagNames(call *Call) []Finding {
	insensitive := call.Bool(uniqueTagNamesCaseOption)
	entries := seq.FlatMapPairs(
		oas.Items(oas.Get(call.Root, "tags")),
		func(i int, tag *yaml.Node) []uniqueTagNamesEntry {
			name := oas.Get(tag, "name")
			if !oas.IsScalar(name) {
				return nil
			}

			if insensitive {
				return []uniqueTagNamesEntry{{index: i, name: name, key: Fold(name.Value)}}
			}

			return []uniqueTagNamesEntry{{index: i, name: name, key: name.Value}}
		},
	)

	return seq.FlatMapIndexed(entries, func(position int, entry uniqueTagNamesEntry) []Finding {
		first := slices.IndexFunc(entries, func(other uniqueTagNamesEntry) bool {
			return other.key == entry.key
		})

		return When(first != position, func() Finding {
			return Finding{
				Node:    entry.name,
				Path:    oas.Join(oas.Index("$.tags", entry.index), "name"),
				Message: "tag name " + entry.name.Value + " is duplicated",
			}
		})
	})
}
