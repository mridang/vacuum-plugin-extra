package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// EmptyRequiredOrEnum returns the extra-empty-required-or-enum rule.
//
// # What it checks
//
// In every schema the required and enum lists, when present as lists, must
// contain at least one entry and must not contain duplicates. An empty enum
// can never be satisfied, an empty required list is invalid in OpenAPI 3.0 and
// pointless in 3.1, and duplicated entries are always a copy and paste slip.
// The boolean required flag used by parameters and request bodies is not a
// list and is not examined.
//
// # Where it comes from
//
// KaiZen OpenApi Parser, whose schema validation reports empty and
// duplicated required and enum lists.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      required: []
//	      properties:
//	        kind:
//	          type: string
//	          enum: [cat, cat]
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      required: [kind]
//	      properties:
//	        kind:
//	          type: string
//	          enum: [cat, dog]
//
// # Known false positives
//
// None known. A property literally named required or enum is a mapping inside
// properties and is not mistaken for the keyword.
func EmptyRequiredOrEnum() Rule {
	return New(Spec{
		Name:        "empty-required-or-enum",
		Description: "The required and enum lists of a schema must be non-empty and free of duplicates.",
		HowToFix:    "Remove the empty list or the duplicated entry.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkEmptyRequiredOrEnum),
	})
}

func checkEmptyRequiredOrEnum(_ *Call, schema oas.Schema) []Finding {
	return seq.FlatMapSlice([]string{literalRequired, keywordEnum}, func(keyword string) []Finding {
		list := oas.Get(schema.Node, keyword)
		if !oas.IsSeq(list) {
			return nil
		}

		return emptyRequiredOrEnumList(keyword, list, oas.Join(schema.Path, keyword))
	})
}

func emptyRequiredOrEnumList(keyword string, list *yaml.Node, path string) []Finding {
	if len(list.Content) == 0 {
		return One(Finding{Node: list, Path: path, Message: keyword + " must not be empty"})
	}

	return seq.FlatMapIndexed(list.Content, func(i int, item *yaml.Node) []Finding {
		return When(slices.ContainsFunc(list.Content[:i], func(other *yaml.Node) bool {
			return emptyRequiredOrEnumSame(other, item)
		}), func() Finding {
			return Finding{
				Node:    item,
				Path:    oas.Index(path, i),
				Message: keyword + " contains a duplicate entry",
			}
		})
	})
}

func emptyRequiredOrEnumSame(a, b *yaml.Node) bool {
	if a.Kind != b.Kind || a.Value != b.Value || len(a.Content) != len(b.Content) {
		return false
	}

	if oas.IsScalar(a) && a.ShortTag() != b.ShortTag() {
		return false
	}

	for i := range a.Content {
		if !emptyRequiredOrEnumSame(a.Content[i], b.Content[i]) {
			return false
		}
	}

	return true
}
