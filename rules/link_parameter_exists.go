package rules

import (
	"net/url"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// LinkParameterExists returns the extra-link-parameter-exists rule.
//
// # What it checks
//
// A port of the link validation of the KaiZen OpenApi Parser (LinkValidator),
// run against every link in components.links, components.responses and the
// responses of every operation:
//
//   - a link must name its target with exactly one of operationId and
//     operationRef (neither, or both, is an error);
//   - an operationId must identify an operation of the document;
//   - a local operationRef (#/paths/~1pets~1{id}/get, percent-encoding is
//     decoded) must point at an operation of the document;
//   - every key of the link parameters map must name a parameter of the target
//     operation, including parameters inherited from the path item. A key may
//     be qualified with its location (path.id, query.limit) or be bare (id). A
//     bare key that matches parameters in more than one location of the target
//     (a query and a header parameter of the same name) is ambiguous and is
//     reported, as KaiZen's "name appears more than once" warning does;
//   - an operationRef that is not local (another file or URL) cannot be
//     followed. KaiZen warns that operationRef is not supported; this rule
//     reports it unless the option is switched off, and then checks nothing
//     else for that link.
//
// Links that are a $ref are checked where they are defined.
//
// # Where it comes from
//
// KaiZen OpenApi Parser, val3/LinkValidator.java (messages NoOpIdNoOpRefInLink,
// OpIdAndOpRefInLink, OpIdNotFound, OpPathNotFound, BadLinkParam,
// AmbigLinkParam and OperationRefUnSupp).
//
// # Options
//
//   - reportRemoteOperationRef: report links whose operationRef is not local,
//     because their parameters cannot be verified. Default: true.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      operationId: getPet
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//	      responses:
//	        "200":
//	          description: ok
//	          links:
//	            self:
//	              operationId: getPet
//	              parameters:
//	                petId: $response.body#/id
//
// # Good
//
//	self:
//	  operationId: getPet
//	  parameters:
//	    id: $response.body#/id
//
// # Known false positives and limitations
//
// Only local references are resolved, so links to operations in external files
// are not checked (see the option). Parameters that the target inherits from an
// unresolved external path item are not seen.
func LinkParameterExists() Rule {
	return New(Spec{
		Name:        "link-parameter-exists",
		Description: "Links must identify an existing operation and use parameters that exist in it.",
		HowToFix:    "Give the link exactly one valid operationId or operationRef and rename its parameters to match the target operation.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        linkParameterExistsRemoteOption,
				Description: "Report links whose operationRef is not local, because they cannot be verified.",
				Default:     true,
			},
		},
		Check: OnDocument(checkLinkParameterExists),
	})
}

const linkParameterExistsRemoteOption = "reportRemoteOperationRef"

type linkParameterExistsLink struct {
	node *yaml.Node
	path string
}

func checkLinkParameterExists(call *Call) []Finding {
	return seq.FlatMapSlice(linkParameterExistsLinks(call.Root), func(link linkParameterExistsLink) []Finding {
		target, problems, ok := linkParameterExistsTarget(call, link)
		if !ok {
			return problems
		}

		return slices.Concat(problems, linkParameterExistsParameters(call.Root, link, target))
	})
}

func linkParameterExistsParameters(root *yaml.Node, link linkParameterExistsLink, target oas.Operation) []Finding {
	available := oas.Parameters(root, target)
	path := oas.Join(link.path, "parameters")

	return seq.FlatMapPairs(oas.Pairs(oas.Get(link.node, "parameters")), func(key, _ *yaml.Node) []Finding {
		message := linkParameterExistsProblem(key.Value, available)

		return When(message != "", func() Finding {
			return Finding{Node: key, Path: oas.Join(path, key.Value), Message: message}
		})
	})
}

func linkParameterExistsProblem(name string, available []oas.Parameter) string {
	if location, bare, ok := strings.Cut(name, "."); ok && linkParameterExistsIsLocation(location) {
		if slices.ContainsFunc(available, func(parameter oas.Parameter) bool {
			return parameter.In() == location && parameter.Name() == bare
		}) {
			return ""
		}

		return "link parameter " + name + " does not exist in the target operation"
	}

	matches := len(seq.Filter(slices.Values(available), func(parameter oas.Parameter) bool {
		return parameter.Name() == name
	}))

	switch {
	case matches == 0:
		return "link parameter " + name + " does not exist in the target operation"
	case matches > 1:
		return "link parameter " + name + " is ambiguous; qualify it with its location"
	default:
		return ""
	}
}

func linkParameterExistsIsLocation(text string) bool {
	switch text {
	case literalPath, literalQuery, keywordHeader, "cookie":
		return true
	default:
		return false
	}
}

func linkParameterExistsCollect(container *yaml.Node, path string) []linkParameterExistsLink {
	return seq.FlatMapPairs(oas.Pairs(container), func(key, value *yaml.Node) []linkParameterExistsLink {
		if !oas.IsMap(value) || oas.Has(value, "$ref") {
			return nil
		}

		return []linkParameterExistsLink{{node: value, path: oas.Join(path, key.Value)}}
	})
}

func linkParameterExistsLinks(root *yaml.Node) []linkParameterExistsLink {
	return slices.Concat(
		linkParameterExistsCollect(oas.Dig(root, "components", "links"), "$.components.links"),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(key, response *yaml.Node) []linkParameterExistsLink {
				return linkParameterExistsCollect(
					oas.Get(response, "links"),
					oas.Join(oas.JoinAll("$.components.responses", key.Value), "links"),
				)
			},
		),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []linkParameterExistsLink {
			return seq.FlatMapPairs(
				oas.Pairs(oas.Get(operation.Node, "responses")),
				func(key, response *yaml.Node) []linkParameterExistsLink {
					return linkParameterExistsCollect(
						oas.Get(response, "links"),
						oas.Join(oas.JoinAll(operation.Path, "responses", key.Value), "links"),
					)
				},
			)
		}),
	)
}

// linkParameterExistsTarget resolves the operation a link points at. It returns
// the findings about the link itself and whether a target operation was found.
func linkParameterExistsTarget(call *Call, link linkParameterExistsLink) (oas.Operation, []Finding, bool) {
	id := oas.StringAt(link.node, "operationId")
	ref := oas.StringAt(link.node, "operationRef")

	switch {
	case id == "" && ref == "":
		return oas.Operation{}, linkParameterExistsAt(
			link.node,
			link.path,
			"link must contain either 'operationRef' or 'operationId'",
		), false
	case id != "" && ref != "":
		return oas.Operation{}, linkParameterExistsAt(
			link.node,
			link.path,
			"link may not contain both 'operationRef' and 'operationId'",
		), false
	case id != "":
		return linkParameterExistsByID(call.Root, link, id)
	default:
		return linkParameterExistsByRef(call, link, ref)
	}
}

func linkParameterExistsAt(node *yaml.Node, path, message string) []Finding {
	return []Finding{{Node: node, Path: path, Message: message}}
}

func linkParameterExistsByID(
	root *yaml.Node,
	link linkParameterExistsLink,
	id string,
) (oas.Operation, []Finding, bool) {
	for operation := range oas.Operations(root) {
		if oas.StringAt(operation.Node, "operationId") == id {
			return operation, nil, true
		}
	}

	_, value := oas.Entry(link.node, "operationId")

	return oas.Operation{}, linkParameterExistsAt(value, oas.Join(link.path, "operationId"),
		"operationId in link does not identify an operation in the containing document"), false
}

func linkParameterExistsDecode(ref string) string {
	decoded, err := url.PathUnescape(ref)
	if err != nil {
		return ref
	}

	return decoded
}

func linkParameterExistsByRef(call *Call, link linkParameterExistsLink, ref string) (oas.Operation, []Finding, bool) {
	_, value := oas.Entry(link.node, "operationRef")
	path := oas.Join(link.path, "operationRef")

	if !strings.HasPrefix(ref, "#") {
		if !call.Bool(linkParameterExistsRemoteOption) {
			return oas.Operation{}, nil, false
		}

		return oas.Operation{}, linkParameterExistsAt(value, path,
			"operationRef is not local, so the link parameters cannot be verified"), false
	}

	target := oas.Pointer(call.Root, linkParameterExistsDecode(ref))
	for operation := range oas.Operations(call.Root) {
		if target != nil && operation.Node == target {
			return operation, nil, true
		}
	}

	return oas.Operation{}, linkParameterExistsAt(value, path,
		"operationRef in link does not identify an operation in the containing document"), false
}
