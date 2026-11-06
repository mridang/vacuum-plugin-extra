package rules

import (
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	reservedHeaderParameterContentType = "Content-Type"
	reservedHeaderParameterHeader      = "header"
)

const reservedHeaderParameterAuthorization = "Authorization"

const reservedHeaderParameterAccept = "Accept"

const reservedHeaderParameterNames = "names"

// ReservedHeaderParameter returns the extra-reserved-header-parameter rule.
//
// # What it checks
//
// A parameter with in: header must not be named after a header that OpenAPI
// reserves. The names are compared case-insensitively and default to Accept,
// Content-Type and Authorization. Parameters are looked at on path items, on
// operations and under components.parameters. A response header named
// Content-Type is flagged too, since OpenAPI ignores it as well.
//
// # Why
//
// The OpenAPI specification says that header parameters named Accept,
// Content-Type or Authorization SHALL be ignored, and that a Content-Type
// response header SHALL be ignored. These headers are controlled by the
// requestBody and response content keys (Content-Type, Accept) and by security
// schemes (Authorization). Declaring them as parameters is misleading: some
// tools honour the declaration and override the media type, others silently
// drop it, so the contract and the generated client disagree.
//
// # Where it comes from
//
// specmatic (it overrides the media type from a Content-Type header parameter
// and warns when Accept header values cannot match the response content types).
// This rule reports the declaration itself and, in addition, the two
// situations specmatic warns about (the overridden media type and the Accept
// header that does not allow a response content type).
//
// # Options
//
//   - names (list of strings, default Accept, Content-Type, Authorization): the
//     reserved header names.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      parameters:
//	        - name: Content-Type
//	          in: header
//	          schema: {type: string}
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema: {type: object}
//
// # Known false positives and limitations
//
// Authorization is reserved by the specification but widely declared by
// documents whose security scheme cannot express their token; remove it from
// names to allow it. Parameters given by $ref are checked where they are
// declared in components.parameters. Accept and Content-Type values are only
// known for enum and const schemas, as in specmatic; a header schema that is
// a free string is never compared. The Accept and Content-Type compatibility
// checks apply even when the names option does not list those headers.
func ReservedHeaderParameter() Rule {
	return New(Spec{
		Name:        "reserved-header-parameter",
		Description: "Header parameters must not use the reserved names Accept, Content-Type or Authorization.",
		HowToFix:    "Remove the parameter; describe the media type with content keys and authentication with security schemes.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        reservedHeaderParameterNames,
				Description: "Header names that must not be declared as parameters.",
				Default: []any{
					reservedHeaderParameterAccept,
					reservedHeaderParameterContentType,
					reservedHeaderParameterAuthorization,
				},
			},
		},
		Check: OnDocument(checkReservedHeaderParameter),
	})
}

// checkReservedHeaderParameter reports header parameters with reserved names.
func checkReservedHeaderParameter(call *Call) []Finding {
	names := seq.Map(slices.Values(call.Strings(reservedHeaderParameterNames)), Fold)

	list := func(parameters *yaml.Node, path string) []Finding {
		return seq.FlatMapPairs(oas.Items(parameters), func(i int, parameter *yaml.Node) []Finding {
			return reservedHeaderParameterCheck(parameter, oas.Index(path, i), names)
		})
	}

	return slices.Concat(
		seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
			return list(oas.Get(item.Node, "parameters"), oas.Join(item.Path, "parameters"))
		}),
		seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
			return slices.Concat(
				list(oas.Get(operation.Node, "parameters"), oas.Join(operation.Path, "parameters")),
				reservedHeaderParameterResponses(operation, names),
				reservedHeaderParameterCompatibility(call.Root, operation),
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(call.Root, "components", "parameters")),
			func(name, parameter *yaml.Node) []Finding {
				return reservedHeaderParameterCheck(
					parameter,
					oas.JoinAll("$.components.parameters", name.Value),
					names,
				)
			},
		),
	)
}

// reservedHeaderParameterCheck checks one parameter object.
func reservedHeaderParameterCheck(parameter *yaml.Node, path string, names []string) []Finding {
	if oas.StringAt(parameter, "in") != reservedHeaderParameterHeader {
		return nil
	}

	key, name := oas.Entry(parameter, "name")

	return When(oas.IsScalar(name) && slices.Contains(names, Fold(name.Value)), func() Finding {
		return Finding{
			Node:    name,
			Path:    oas.Join(path, key.Value),
			Message: "header parameter '" + name.Value + "' is reserved by OpenAPI and is ignored; use content, security schemes or remove it",
		}
	})
}

// reservedHeaderParameterResponses flags Content-Type response headers of an operation.
func reservedHeaderParameterResponses(operation oas.Operation, names []string) []Finding {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(operation.Node, "responses")),
		func(code, response *yaml.Node) []Finding {
			return seq.FlatMapPairs(oas.Pairs(oas.Get(response, "headers")), func(name, _ *yaml.Node) []Finding {
				return When(
					strings.EqualFold(name.Value, reservedHeaderParameterContentType) &&
						slices.Contains(names, Fold(name.Value)),
					func() Finding {
						return Finding{
							Node:    name,
							Path:    oas.JoinAll(operation.Path, "responses", code.Value, "headers", name.Value),
							Message: "response header 'Content-Type' is reserved by OpenAPI and is ignored; use content instead",
						}
					},
				)
			})
		},
	)
}

const (
	reservedHeaderParameterQuality     = "q="
	reservedHeaderParameterFullRange   = "*"
	reservedHeaderParameterDefaultSpec = 1.0
)

// reservedHeaderParameterCompatibility reports Content-Type overrides and
// Accept headers that do not allow a response content type.
func reservedHeaderParameterCompatibility(root *yaml.Node, operation oas.Operation) []Finding {
	parameters := oas.Parameters(root, operation)
	contentTypes := reservedHeaderParameterLastValues(root, parameters, reservedHeaderParameterContentType, true)
	accepts := reservedHeaderParameterLastValues(root, parameters, reservedHeaderParameterAccept, false)
	body := oas.Resolve(root, oas.Get(operation.Node, "requestBody"))

	return slices.Concat(
		reservedHeaderParameterOverride(
			contentTypes, oas.Get(body, "content"), oas.JoinAll(operation.Path, "requestBody", "content")),
		seq.FlatMapPairs(oas.Pairs(oas.Get(operation.Node, "responses")), func(code, raw *yaml.Node) []Finding {
			response := oas.Resolve(root, raw)
			path := reservedHeaderParameterResponsePath(operation, code.Value, raw)
			header := reservedHeaderParameterHeaderValues(root, response)

			return slices.Concat(
				reservedHeaderParameterOverride(header, oas.Get(response, "content"), oas.Join(path, "content")),
				reservedHeaderParameterAccepted(accepts, oas.Get(response, "content"), oas.Join(path, "content")),
			)
		}),
	)
}

// reservedHeaderParameterLastValues returns the known values of the last header parameter with the
// given name. When single is set, only parameters with exactly one known value count.
func reservedHeaderParameterLastValues(
	root *yaml.Node,
	parameters []oas.Parameter,
	name string,
	single bool,
) []string {
	for _, parameter := range slices.Backward(parameters) {
		if parameter.In() != reservedHeaderParameterHeader || !strings.EqualFold(parameter.Name(), name) {
			continue
		}

		values := reservedHeaderParameterKnown(root, oas.Get(parameter.Node, "schema"))
		if !single || len(values) == 1 {
			return values
		}
	}

	return nil
}

// reservedHeaderParameterResponsePath returns the location of a response
// object, following a reference into components.responses.
func reservedHeaderParameterResponsePath(operation oas.Operation, code string, raw *yaml.Node) string {
	if name, ok := strings.CutPrefix(oas.Ref(raw), "#/components/responses/"); ok {
		return oas.JoinAll("$.components.responses", name)
	}

	return oas.JoinAll(operation.Path, "responses", code)
}

// reservedHeaderParameterHeaderValues returns the single possible Content-Type
// header value of a response, or nil.
func reservedHeaderParameterHeaderValues(root, response *yaml.Node) []string {
	for name, header := range oas.Pairs(oas.Get(response, "headers")) {
		if !strings.EqualFold(name.Value, reservedHeaderParameterContentType) {
			continue
		}

		values := reservedHeaderParameterKnown(root, oas.Get(oas.Resolve(root, header), "schema"))
		if len(values) == 1 {
			return values
		}
	}

	return nil
}

// reservedHeaderParameterKnown returns the string values a header schema can
// take when they are known: an enum of strings or a const. Otherwise nil.
func reservedHeaderParameterKnown(root, schema *yaml.Node) []string {
	resolved := oas.Resolve(root, schema)

	if constant := oas.Get(resolved, "const"); oas.IsScalar(constant) {
		return []string{constant.Value}
	}

	enum := oas.Get(resolved, "enum")
	values := oas.Strings(enum)

	if len(values) == 0 || len(values) != len(enum.Content) {
		return nil
	}

	return values
}

// reservedHeaderParameterOverride reports media type keys that the single
// Content-Type header value overrides.
func reservedHeaderParameterOverride(values []string, content *yaml.Node, path string) []Finding {
	if len(values) != 1 {
		return nil
	}

	return seq.FlatMapSlice(oas.MediaTypes(content, path), func(media oas.MediaType) []Finding {
		return When(media.Name != values[0], func() Finding {
			return Finding{
				Node: media.Node,
				Path: media.Path,
				Message: "Media type \"" + media.Name + "\" does not match the respective Content-Type header \"" +
					values[0] + "\". Using the Content-Type header as an override",
			}
		})
	})
}

// reservedHeaderParameterAccepted reports response media types that none of
// the known Accept values allows.
func reservedHeaderParameterAccepted(accepts []string, content *yaml.Node, path string) []Finding {
	if len(accepts) == 0 {
		return nil
	}

	return seq.FlatMapSlice(oas.MediaTypes(content, path), func(media oas.MediaType) []Finding {
		allowed := slices.ContainsFunc(accepts, func(accept string) bool {
			return reservedHeaderParameterAccepts(accept, media.Name)
		})

		return When(!allowed, func() Finding {
			return Finding{
				Node: media.Node,
				Path: media.Path,
				Message: "Accept header values " + strings.Join(accepts, ", ") +
					" do not allow response Content-Type \"" + media.Name + "\"",
			}
		})
	})
}

// reservedHeaderParameterRange is a media range of an Accept header.
type reservedHeaderParameterRange struct {
	kind    string
	subtype string
}

// reservedHeaderParameterAccepts reports whether the Accept header value allows
// the response media type (specmatic isResponseContentTypeAccepted).
func reservedHeaderParameterAccepts(accept, responseType string) bool {
	base, _, _ := strings.Cut(responseType, ";")

	response, ok := reservedHeaderParameterParseRange(base)
	if !ok {
		return true
	}

	for token := range strings.SplitSeq(accept, ",") {
		accepted, quality, valid := reservedHeaderParameterParseToken(token)
		if !valid || quality <= 0 {
			continue
		}

		wildcard := accepted.kind == reservedHeaderParameterFullRange &&
			accepted.subtype == reservedHeaderParameterFullRange
		if wildcard || (accepted.kind == response.kind &&
			(accepted.subtype == reservedHeaderParameterFullRange || accepted.subtype == response.subtype)) {
			return true
		}
	}

	return false
}

// reservedHeaderParameterParseToken parses one Accept token with its q value.
func reservedHeaderParameterParseToken(token string) (reservedHeaderParameterRange, float64, bool) {
	parts := strings.Split(token, ";")

	media, ok := reservedHeaderParameterParseRange(parts[0])
	if !ok {
		return media, 0, false
	}

	for _, part := range parts[1:] {
		trimmed := strings.TrimSpace(part)
		if len(trimmed) >= len(reservedHeaderParameterQuality) &&
			strings.EqualFold(trimmed[:len(reservedHeaderParameterQuality)], reservedHeaderParameterQuality) {
			if quality, err := strconv.ParseFloat(
				strings.TrimSpace(trimmed[len(reservedHeaderParameterQuality):]),
				64,
			); err == nil {
				return media, quality, true
			}

			break
		}
	}

	return media, reservedHeaderParameterDefaultSpec, true
}

// reservedHeaderParameterParseRange parses type/subtype in lower case.
func reservedHeaderParameterParseRange(text string) (reservedHeaderParameterRange, bool) {
	rawKind, rawSubtype, found := strings.Cut(strings.ToLower(strings.TrimSpace(text)), "/")
	kind, subtype := strings.TrimSpace(rawKind), strings.TrimSpace(rawSubtype)

	return reservedHeaderParameterRange{kind: kind, subtype: subtype}, found && kind != "" && subtype != ""
}
