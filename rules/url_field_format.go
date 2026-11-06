package rules

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// URLFieldFormat returns the extra-url-field-format rule.
//
// # What it checks
//
// Every field of the document that holds a URL must contain a well formed one.
// The fields that are inspected are:
//
//   - info.termsOfService, info.contact.url and info.license.url;
//   - the url of externalDocs on the document, on tags and on operations;
//   - the url of every server (variables in braces are replaced by a
//     placeholder before parsing, and a relative URL is accepted because the
//     specification allows it);
//   - authorizationUrl, tokenUrl and refreshUrl of OAuth flows, and
//     openIdConnectUrl of security schemes;
//   - xml.namespace of schemas.
//
// A value is reported when it cannot be parsed as a URL, contains whitespace,
// or (for every field except server urls and OAuth urls, which may be relative)
// is not absolute, meaning it has no scheme.
//
// Tools follow these links, render them as hyperlinks or hand them to HTTP
// clients, so a typo or a relative link in a place that needs an absolute one
// breaks them.
//
// # Where it comes from
//
// KaiZen OpenAPI Parser, which validates URL typed fields.
//
// # Options
//
// None.
//
// # Bad
//
//	info:
//	  contact:
//	    url: example.com/support
//
// # Good
//
//	info:
//	  contact:
//	    url: https://example.com/support
//
// # Known false positives
//
// A server URL with a template that stands for more than a host part, for
// example a whole URL held in a variable, is accepted without being checked
// further. Only the first url of a field is checked; no network access is made
// to see whether the address exists.
func URLFieldFormat() Rule {
	return New(Spec{
		Name:        "url-field-format",
		Description: "URL fields must hold well formed URLs.",
		HowToFix:    "Correct the URL, and make it absolute where the field requires it.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkURLFieldFormat),
	})
}

// urlFieldFormatField is one URL field and how strictly it is checked.
type urlFieldFormatField struct {
	node     *yaml.Node
	key      string
	path     string
	relative bool
}

func checkURLFieldFormat(call *Call) []Finding {
	return seq.FlatMapSlice(urlFieldFormatFields(call.Root), func(field urlFieldFormatField) []Finding {
		return When(!urlFieldFormatValid(field), func() Finding {
			return Finding{
				Node:    field.node,
				Path:    oas.Join(field.path, field.key),
				Message: field.key + " must be a well formed " + urlFieldFormatKind(field) + " URL",
			}
		})
	})
}

func urlFieldFormatKind(field urlFieldFormatField) string {
	if field.relative {
		return "relative or absolute"
	}

	return "absolute"
}

func urlFieldFormatFields(root *yaml.Node) []urlFieldFormatField {
	return slices.Concat(
		urlFieldFormatAt(oas.Get(root, "info"), "$.info", false, "termsOfService"),
		urlFieldFormatAt(oas.Dig(root, "info", "contact"), "$.info.contact", false, "url"),
		urlFieldFormatAt(oas.Dig(root, "info", "license"), "$.info.license", false, "url"),
		urlFieldFormatAt(oas.Get(root, "externalDocs"), "$.externalDocs", false, "url"),
		seq.FlatMapPairs(oas.Items(oas.Get(root, "servers")), func(i int, server *yaml.Node) []urlFieldFormatField {
			return urlFieldFormatAt(server, oas.Index("$.servers", i), true, "url")
		}),
		seq.FlatMapPairs(oas.Items(oas.Get(root, "tags")), func(i int, tag *yaml.Node) []urlFieldFormatField {
			return urlFieldFormatAt(
				oas.Get(tag, "externalDocs"),
				oas.Join(oas.Index("$.tags", i), "externalDocs"),
				false,
				"url",
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []urlFieldFormatField {
			return urlFieldFormatAt(
				oas.Get(operation.Node, "externalDocs"),
				oas.Join(operation.Path, "externalDocs"),
				false,
				"url",
			)
		}),
		urlFieldFormatSecurity(root),
		seq.FlatMap(oas.Schemas(root), func(schema oas.Schema) []urlFieldFormatField {
			return urlFieldFormatAt(oas.Get(schema.Node, "xml"), oas.Join(schema.Path, "xml"), false, "namespace")
		}),
	)
}

func urlFieldFormatSecurity(root *yaml.Node) []urlFieldFormatField {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Dig(root, "components", "securitySchemes")),
		func(name, scheme *yaml.Node) []urlFieldFormatField {
			path := oas.JoinAll("$.components.securitySchemes", name.Value)

			return slices.Concat(
				urlFieldFormatAt(scheme, path, false, "openIdConnectUrl"),
				seq.FlatMapPairs(
					oas.Pairs(oas.Get(scheme, "flows")),
					func(flowName, flow *yaml.Node) []urlFieldFormatField {
						flowPath := oas.JoinAll(path, "flows", flowName.Value)

						return seq.FlatMapSlice(
							[]string{"authorizationUrl", "tokenUrl", "refreshUrl"},
							func(key string) []urlFieldFormatField {
								return urlFieldFormatAt(flow, flowPath, true, key)
							},
						)
					},
				),
			)
		},
	)
}

func urlFieldFormatAt(owner *yaml.Node, path string, relative bool, key string) []urlFieldFormatField {
	value := oas.Get(owner, key)
	if !oas.IsScalar(value) {
		return nil
	}

	return []urlFieldFormatField{{node: value, key: key, path: path, relative: relative}}
}

func urlFieldFormatTemplate() *regexp.Regexp {
	return regexp.MustCompile(`\{[^{}]*\}`)
}

func urlFieldFormatValid(field urlFieldFormatField) bool {
	text := strings.TrimSpace(field.node.Value)
	if text == "" || strings.ContainsFunc(text, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' }) {
		return false
	}

	parsed, err := url.Parse(urlFieldFormatTemplate().ReplaceAllString(text, "x"))
	if err != nil {
		return false
	}

	return field.relative || parsed.Scheme != ""
}
