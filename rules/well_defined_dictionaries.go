package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	wellDefinedDictionariesAllOf         = "allOf"
	wellDefinedDictionariesAdditional    = "additionalProperties"
	wellDefinedDictionariesType          = "type"
	wellDefinedDictionariesContains      = "contains"
	wellDefinedDictionariesPrefixItems   = "prefixItems"
	wellDefinedDictionariesPropertyNames = "propertyNames"
	wellDefinedDictionariesElse          = "else"
	wellDefinedDictionariesDefs          = "$defs"
	wellDefinedDictionariesValueType     = "requireValueType"
	wellDefinedDictionariesNested        = "forbidNestedDictionaries"
	wellDefinedDictionariesDefinition    = "requireObjectDefinition"
	wellDefinedDictionariesMaxDepth      = 32
	wellDefinedDictionariesBoolTag       = "!!bool"
	wellDefinedDictionariesNullTag       = "!!null"
	wellDefinedDictionariesIntTag        = "!!int"
	wellDefinedDictionariesFloatTag      = "!!float"
	wellDefinedDictionariesMsgBoth       = "Object schemas must be either a model or a dictionary - they cannot be both"
	wellDefinedDictionariesMsgAmbigous   = "Dictionary schemas must have a single, well-defined value type"
	wellDefinedDictionariesMsgNested     = "Dictionaries must not have values that are also dictionaries"
	wellDefinedDictionariesMsgNone       = "Object schemas must define either properties, or (additional/pattern)Properties " +
		"with a concrete type"
)

// WellDefinedDictionaries returns the extra-well-defined-dictionaries rule.
//
// # What it checks
//
// A faithful port of the IBM function wellDefinedDictionaries. For every
// object schema (one with type object, or without a type but with properties,
// additionalProperties or patternProperties; composition is followed as in IBM
// schemaHasConstraint: an allOf member suffices, oneOf and anyOf need every
// member) that is not itself the value schema of a dictionary:
//
//   - a schema that is not a dictionary and defines no properties is reported
//     ("Object schemas must define either properties, or
//     (additional/pattern)Properties with a concrete type"), controlled by
//     requireObjectDefinition;
//   - a dictionary (additionalProperties other than false, or
//     patternProperties) that also defines properties is reported ("they cannot
//     be both");
//   - with requireValueType, a dictionary whose value schema (every
//     patternProperties value is checked) is not an object or does not define a
//     type, directly or through composition, is reported, so
//     additionalProperties: true and {} are reported;
//   - with forbidNestedDictionaries, a dictionary whose value schema is itself
//     a dictionary is reported.
//
// Findings are reported at the object schema, as IBM does. $ref values are
// resolved. Members of allOf, oneOf, anyOf and not are not evaluated on their
// own; only the schema that composes them is.
//
// # Why
//
// A schema that mixes fixed properties with free-form additional properties is
// ambiguous: generators cannot decide between a class and a map, and clients
// lose the type of the extra values. Untyped values and dictionaries of
// dictionaries are hard to model in strongly typed languages and to evolve
// compatibly.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-well-defined-dictionaries).
//
// # Options
//
//   - requireValueType (bool, default true): require a typed value schema.
//   - forbidNestedDictionaries (bool, default true): reject dictionaries whose
//     values are dictionaries.
//   - requireObjectDefinition (bool, default true): report object schemas that
//     define neither properties nor a dictionary.
//
// # Bad
//
//	components:
//	  schemas:
//	    Labels:
//	      type: object
//	      properties:
//	        name: {type: string}
//	      additionalProperties:
//	        type: string
//	    Free:
//	      type: object
//
// # Good
//
//	components:
//	  schemas:
//	    Labels:
//	      type: object
//	      additionalProperties:
//	        type: string
//
// # Known false positives and limitations
//
// IBM only evaluates schemas reachable from operations and parameters, while
// this rule also evaluates every schema under components.schemas, so an unused
// component can be reported. Only local $ref values are followed.
func WellDefinedDictionaries() Rule {
	return New(Spec{
		Name:        "well-defined-dictionaries",
		Description: "A schema must be either a model or a dictionary, and dictionaries need a well-defined value type.",
		HowToFix:    "Remove either properties or additionalProperties, and give the dictionary values a single type.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        wellDefinedDictionariesValueType,
				Description: "Require dictionary values to declare a type.",
				Default:     true,
			},
			{Name: wellDefinedDictionariesNested, Description: "Reject dictionaries of dictionaries.", Default: true},
			{
				Name:        wellDefinedDictionariesDefinition,
				Description: "Report object schemas that define neither properties nor a dictionary.",
				Default:     true,
			},
		},
		Check: OnDocument(checkWellDefinedDictionaries),
	})
}

// checkWellDefinedDictionaries inspects every object schema that is evaluated
// on its own: not a composition member, negation, dictionary value or other
// sub schema that IBM skips.
func checkWellDefinedDictionaries(call *Call) []Finding {
	schemas := slices.Collect(oas.Schemas(call.Root))
	skipped := seq.FlatMapSlice(schemas, func(schema oas.Schema) []*yaml.Node {
		return wellDefinedDictionariesNonRoots(schema.Node)
	})

	return seq.FlatMapSlice(schemas, func(schema oas.Schema) []Finding {
		if slices.Contains(skipped, schema.Node) || oas.Ref(schema.Node) != "" || oas.UnderNegation(schema.Path) {
			return nil
		}

		return wellDefinedDictionariesCheck(call, schema)
	})
}

// wellDefinedDictionariesNonRoots returns the sub schemas that IBM does not evaluate on their own: members of
// composition keywords, negations, dictionary values and the keywords validateNestedSchemas does not descend into.
func wellDefinedDictionariesNonRoots(schema *yaml.Node) []*yaml.Node {
	members := seq.FlatMapSlice([]string{
		wellDefinedDictionariesAllOf,
		keywordAnyOf,
		literalOneof,
		wellDefinedDictionariesPrefixItems,
	}, func(key string) []*yaml.Node {
		return wellDefinedDictionariesMembers(oas.Get(schema, key))
	})
	single := seq.FlatMapSlice([]string{
		wellDefinedDictionariesAdditional,
		keywordNot,
		wellDefinedDictionariesContains,
		wellDefinedDictionariesPropertyNames,
		"if",
		"then",
		wellDefinedDictionariesElse,
		keywordUnevaluatedProperties,
		"unevaluatedItems",
	}, func(key string) []*yaml.Node {
		return []*yaml.Node{oas.Get(schema, key)}
	})
	values := seq.FlatMapSlice(
		[]string{literalPatternproperties, "dependentSchemas", wellDefinedDictionariesDefs},
		func(key string) []*yaml.Node {
			return wellDefinedDictionariesValues(oas.Get(schema, key))
		},
	)

	return slices.Concat(members, single, values)
}

// wellDefinedDictionariesMembers returns the items of a sequence.
func wellDefinedDictionariesMembers(list *yaml.Node) []*yaml.Node {
	return seq.FlatMapPairs(oas.Items(list), func(_ int, item *yaml.Node) []*yaml.Node {
		return []*yaml.Node{item}
	})
}

// wellDefinedDictionariesValues returns the values of a mapping.
func wellDefinedDictionariesValues(mapping *yaml.Node) []*yaml.Node {
	return seq.FlatMapPairs(oas.Pairs(mapping), func(_, value *yaml.Node) []*yaml.Node {
		return []*yaml.Node{value}
	})
}

// wellDefinedDictionariesCheck applies the IBM checks to one object schema.
func wellDefinedDictionariesCheck(call *Call, schema oas.Schema) []Finding {
	root := call.Root
	if !wellDefinedDictionariesHas(root, schema.Node, wellDefinedDictionariesIsObject, 0) {
		return nil
	}

	report := func(message string) Finding {
		return Finding{Node: schema.Node, Path: schema.Path, Message: message}
	}
	both := wellDefinedDictionariesDefines(root, schema.Node, literalProperties)

	if !wellDefinedDictionariesDictionary(root, schema.Node) {
		return When(call.Bool(wellDefinedDictionariesDefinition) && !both, func() Finding {
			return report(wellDefinedDictionariesMsgNone)
		})
	}

	return slices.Concat(
		When(both, func() Finding {
			return report(wellDefinedDictionariesMsgBoth)
		}),
		When(
			call.Bool(wellDefinedDictionariesValueType) &&
				wellDefinedDictionariesValuesHave(root, schema.Node, wellDefinedDictionariesAmbiguous),
			func() Finding {
				return report(wellDefinedDictionariesMsgAmbigous)
			},
		),
		When(
			call.Bool(wellDefinedDictionariesNested) &&
				wellDefinedDictionariesValuesHave(root, schema.Node, wellDefinedDictionariesNestedValue),
			func() Finding {
				return report(wellDefinedDictionariesMsgNested)
			},
		),
	)
}

// wellDefinedDictionariesIsObject mirrors the IBM object schema predicate for a single schema.
func wellDefinedDictionariesIsObject(schema *yaml.Node) bool {
	if oas.Has(schema, wellDefinedDictionariesType) {
		return oas.HasType(schema, literalObject)
	}

	additional := oas.Get(schema, wellDefinedDictionariesAdditional)

	return oas.IsMap(oas.Get(schema, literalProperties)) ||
		oas.IsMap(additional) ||
		(oas.IsScalar(additional) && additional.Tag == wellDefinedDictionariesBoolTag && oas.IsTrue(additional)) ||
		oas.IsMap(oas.Get(schema, literalPatternproperties))
}

// wellDefinedDictionariesDictionary reports whether the schema defines additionalProperties or patternProperties.
func wellDefinedDictionariesDictionary(root, schema *yaml.Node) bool {
	return wellDefinedDictionariesDefines(root, schema, wellDefinedDictionariesAdditional) ||
		wellDefinedDictionariesDefines(root, schema, literalPatternproperties)
}

// wellDefinedDictionariesDefines reports whether the schema, through composition, has a truthy field.
func wellDefinedDictionariesDefines(root, schema *yaml.Node, field string) bool {
	return wellDefinedDictionariesHas(root, schema, func(s *yaml.Node) bool {
		return wellDefinedDictionariesTruthy(oas.Get(s, field))
	}, 0)
}

// wellDefinedDictionariesTruthy mirrors JavaScript truthiness for a YAML node.
func wellDefinedDictionariesTruthy(node *yaml.Node) bool {
	if node == nil {
		return false
	}

	if !oas.IsScalar(node) {
		return true
	}

	switch node.Tag {
	case wellDefinedDictionariesNullTag:
		return false
	case wellDefinedDictionariesBoolTag:
		return oas.IsTrue(node)
	case wellDefinedDictionariesIntTag, wellDefinedDictionariesFloatTag:
		number, ok := oas.Number(node)

		return !ok || number != 0
	default:
		return node.Value != ""
	}
}

// wellDefinedDictionariesValuesHave reports whether the dictionary value schemas satisfy a predicate.
func wellDefinedDictionariesValuesHave(root, schema *yaml.Node, predicate func(root, value *yaml.Node) bool) bool {
	return wellDefinedDictionariesHas(root, schema, func(s *yaml.Node) bool {
		if _, additional := oas.Entry(s, wellDefinedDictionariesAdditional); additional != nil {
			return predicate(root, additional)
		}

		return slices.ContainsFunc(
			wellDefinedDictionariesValues(oas.Get(s, literalPatternproperties)),
			func(value *yaml.Node) bool {
				return predicate(root, value)
			},
		)
	}, 0)
}

// wellDefinedDictionariesAmbiguous reports a dictionary value that is not an object or has no type.
func wellDefinedDictionariesAmbiguous(root, value *yaml.Node) bool {
	resolved := oas.Resolve(root, value)

	return !oas.IsMap(resolved) || !wellDefinedDictionariesHas(root, resolved, func(s *yaml.Node) bool {
		return wellDefinedDictionariesTruthy(oas.Get(s, wellDefinedDictionariesType))
	}, 0)
}

// wellDefinedDictionariesNestedValue reports a dictionary value that is itself a dictionary.
func wellDefinedDictionariesNestedValue(root, value *yaml.Node) bool {
	resolved := oas.Resolve(root, value)

	return oas.IsMap(resolved) && wellDefinedDictionariesDictionary(root, resolved)
}

// wellDefinedDictionariesHas mirrors IBM schemaHasConstraint: the predicate holds for the schema, for any allOf
// member, or for every oneOf or anyOf member.
func wellDefinedDictionariesHas(root, schema *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if !oas.IsMap(resolved) || depth > wellDefinedDictionariesMaxDepth {
		return false
	}

	holds := func(member *yaml.Node) bool {
		return wellDefinedDictionariesHas(root, member, predicate, depth+1)
	}

	return predicate(resolved) ||
		slices.ContainsFunc(wellDefinedDictionariesMembers(oas.Get(resolved, wellDefinedDictionariesAllOf)), holds) ||
		slices.ContainsFunc([]string{literalOneof, keywordAnyOf}, func(key string) bool {
			members := wellDefinedDictionariesMembers(oas.Get(resolved, key))

			return len(members) > 0 && !slices.ContainsFunc(members, func(member *yaml.Node) bool {
				return !holds(member)
			})
		})
}
