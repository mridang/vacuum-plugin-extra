package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const encodingKeysMatchPropertiesMaxDepth = 8

// EncodingKeysMatchProperties returns the extra-encoding-keys-match-properties rule.
//
// # What it checks
//
// Each key of a media type object's encoding map must name a property of the
// media type's schema. The schema is resolved if it is a $ref, and properties
// declared in allOf members are included. When the schema is missing, or uses
// anyOf or oneOf (so the property set cannot be known statically), the media
// type is not checked.
//
// # Why
//
// The encoding map describes how individual properties of a multipart or
// form-urlencoded body are serialised. A key that matches no property is a
// typo or a leftover from a renamed property, and is silently ignored by
// tooling, so the intended encoding never applies.
//
// # Where it comes from
//
// KaiZen OpenAPI Parser (media type encoding validation).
//
// # Options
//
// None.
//
// # Bad
//
//	requestBody:
//	  content:
//	    multipart/form-data:
//	      schema:
//	        type: object
//	        properties:
//	          avatar: {type: string, format: binary}
//	      encoding:
//	        avatr:
//	          contentType: image/png
//
// # Good
//
//	encoding:
//	  avatar:
//	    contentType: image/png
//
// # Known false positives and limitations
//
// Schemas that accept arbitrary properties through additionalProperties or
// patternProperties are still checked against their declared properties only.
// Only inline media types of operations and of components.requestBodies and
// components.responses are visited.
func EncodingKeysMatchProperties() Rule {
	return New(Spec{
		Name:        "encoding-keys-match-properties",
		Description: "Encoding map keys must name properties of the media type schema.",
		HowToFix:    "Rename or remove the encoding key so that it matches a property of the schema.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkEncodingKeysMatchProperties),
	})
}

// encodingKeysMatchPropertiesNames holds the property names of a schema and whether they are all known.
type encodingKeysMatchPropertiesNames struct {
	names []string
	known bool
}

// checkEncodingKeysMatchProperties inspects every media type that carries an encoding map.
func checkEncodingKeysMatchProperties(call *Call) []Finding {
	return seq.FlatMapSlice(encodingKeysMatchPropertiesMediaTypes(call.Root), func(media oas.MediaType) []Finding {
		encoding := oas.Get(media.Node, "encoding")
		if !oas.IsMap(encoding) {
			return nil
		}

		properties := encodingKeysMatchPropertiesCollect(call.Root, oas.Get(media.Node, "schema"), 0)
		if !properties.known {
			return nil
		}

		return seq.FlatMapPairs(oas.Pairs(encoding), func(key, _ *yaml.Node) []Finding {
			return When(!slices.Contains(properties.names, key.Value), func() Finding {
				return Finding{
					Node:    key,
					Path:    oas.JoinAll(media.Path, "encoding", key.Value),
					Message: "encoding key '" + key.Value + "' does not match a property of the media type schema",
				}
			})
		})
	})
}

// encodingKeysMatchPropertiesMediaTypes collects the inline media types of the document.
func encodingKeysMatchPropertiesMediaTypes(root *yaml.Node) []oas.MediaType {
	media := func(owner *yaml.Node, path string) []oas.MediaType {
		return oas.MediaTypes(oas.Get(owner, "content"), oas.Join(path, "content"))
	}

	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []oas.MediaType {
			return slices.Concat(
				media(oas.Get(operation.Node, "requestBody"), oas.Join(operation.Path, "requestBody")),
				seq.FlatMapPairs(
					oas.Pairs(oas.Get(operation.Node, "responses")),
					func(key, response *yaml.Node) []oas.MediaType {
						return media(response, oas.JoinAll(operation.Path, "responses", key.Value))
					},
				),
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "requestBodies")),
			func(key, body *yaml.Node) []oas.MediaType {
				return media(body, oas.JoinAll("$.components.requestBodies", key.Value))
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(key, response *yaml.Node) []oas.MediaType {
				return media(response, oas.JoinAll("$.components.responses", key.Value))
			},
		),
	)
}

// encodingKeysMatchPropertiesCollect returns the property names of the schema and whether they are all known.
func encodingKeysMatchPropertiesCollect(root, schema *yaml.Node, depth int) encodingKeysMatchPropertiesNames {
	resolved := oas.Resolve(root, schema)
	if !oas.IsMap(resolved) || depth > encodingKeysMatchPropertiesMaxDepth {
		return encodingKeysMatchPropertiesNames{}
	}

	if oas.Has(resolved, "anyOf") || oas.Has(resolved, "oneOf") {
		return encodingKeysMatchPropertiesNames{}
	}

	members := seq.FlatMapPairs(
		oas.Items(oas.Get(resolved, "allOf")),
		func(_ int, member *yaml.Node) []encodingKeysMatchPropertiesNames {
			return []encodingKeysMatchPropertiesNames{encodingKeysMatchPropertiesCollect(root, member, depth+1)}
		},
	)

	return encodingKeysMatchPropertiesNames{
		names: slices.Concat(
			oas.Keys(oas.Get(resolved, "properties")),
			seq.FlatMapSlice(members, func(member encodingKeysMatchPropertiesNames) []string {
				return member.names
			}),
		),
		known: !slices.ContainsFunc(members, func(member encodingKeysMatchPropertiesNames) bool {
			return !member.known
		}),
	}
}
