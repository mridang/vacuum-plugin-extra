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

// errorContainerSchemaShapeMaxRefDepth bounds the $ref chain followed per schema.
const errorContainerSchemaShapeMaxRefDepth = 16

const (
	errorContainerSchemaShapeArray       = "array"
	errorContainerSchemaShapeProperties  = "properties"
	errorContainerSchemaShapeString      = "string"
	errorContainerSchemaShapeEnum        = "enum"
	errorContainerSchemaShapeMinStatus   = 400
	errorContainerSchemaShapeMaxStatus   = 600
	errorContainerSchemaShapeStatusWidth = 3
)

// ErrorContainerSchemaShape returns the extra-error-container-schema-shape rule.
//
// # What it checks
//
// A port of IBM's ibm-error-response-schemas, which holds error responses to
// the "Error container model" of the IBM API Handbook. Every schema of every
// media type (not only JSON) of a response whose status code is a number from
// 400 to 599 is examined ($ref chains are followed; ranges such as 4XX and
// default are not selected, as in the original). Nothing is merged: properties
// must be declared directly on the schema.
//
// The container:
//
//   - must be an object with properties;
//   - must have a trace property that is a string schema;
//   - may have a status_code property, which must be an integer schema;
//   - must have an errors property that is an array schema whose items are an
//     object with properties (the error model).
//
// The error model:
//
//   - must have a code property that is a string schema with an enum that is an
//     array of the valid error codes;
//   - must have a message property that is a string schema;
//   - must have a more_info property that is a string schema;
//   - may have a target property, which must be an object with properties (the
//     error target model) that has a type property, a string schema whose enum
//     holds exactly field, header and parameter in any order, and a name
//     property that is a string schema.
//
// A schema counts as a string (array, integer) schema through composition like
// in IBM's utilities: its own type, any allOf member, or every oneOf and anyOf
// member; a schema with items counts as an array.
//
// Findings carry the original messages ("Error container model must be an
// object with properties", "Error model must contain property 'code' of type
// string", ...) and the original locations: the properties object when a
// property is missing, the property when it is wrong, the enum for an invalid
// enum. When the schema is a $ref the location is inside the referenced
// component.
//
// # Why
//
// A uniform error body lets clients parse every failure of the service with one
// code path and lets documentation describe the possible error codes.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-error-response-schemas, severity warn).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "404":
//	          description: not found
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	                properties:
//	                  message:
//	                    type: string
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "404":
//	          description: not found
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	                properties:
//	                  trace:
//	                    type: string
//	                  errors:
//	                    type: array
//	                    items:
//	                      type: object
//	                      properties:
//	                        code:
//	                          type: string
//	                          enum: [not_found]
//	                        message:
//	                          type: string
//	                        more_info:
//	                          type: string
//
// # Known false positives and limitations
//
// This is a strict, opinionated shape, the only one the original accepts.
// Services that use RFC 9457 problem details or any other error format are
// flagged for every error response; do not enable the rule for them. Schemas
// composed with allOf are not merged for the object and property checks.
func ErrorContainerSchemaShape() Rule {
	return New(Spec{
		Name:        "error-container-schema-shape",
		Description: "Error response schemas must follow the API Handbook error container model.",
		HowToFix:    "Make the error schema an object with trace and an errors array of error models with code, message and more_info.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkErrorContainerSchemaShape),
	})
}

// errorContainerSchemaShapeChecker resolves references of one document.
type errorContainerSchemaShapeChecker struct {
	root *yaml.Node
}

// errorContainerSchemaShapeStatus reports whether the response code is a number
// from 400 to 599.
func errorContainerSchemaShapeStatus(code string) bool {
	if len(code) != errorContainerSchemaShapeStatusWidth {
		return false
	}

	number, err := strconv.Atoi(code)

	return err == nil && number >= errorContainerSchemaShapeMinStatus && number < errorContainerSchemaShapeMaxStatus
}

// checkErrorContainerSchemaShape validates every 4xx and 5xx response schema
// and reports each path and message once.
func checkErrorContainerSchemaShape(call *Call) []Finding {
	checker := errorContainerSchemaShapeChecker{root: call.Root}

	return errorContainerSchemaShapeUnique(PerResponse(
		func(_ *Call, _ oas.Operation, response oas.Response) []Finding {
			return checker.response(response)
		},
	)(call))
}

// errorContainerSchemaShapeUnique keeps the first finding of every path and
// message.
func errorContainerSchemaShapeUnique(findings []Finding) []Finding {
	return seq.FlatMapIndexed(findings, func(i int, finding Finding) []Finding {
		return When(
			slices.IndexFunc(findings, func(other Finding) bool {
				return other.Path == finding.Path && other.Message == finding.Message
			}) == i,
			func() Finding { return finding },
		)
	})
}

// response checks the media types of one response.
func (c errorContainerSchemaShapeChecker) response(response oas.Response) []Finding {
	if !errorContainerSchemaShapeStatus(response.Code) {
		return nil
	}

	content := oas.Get(response.Node, "content")

	return seq.FlatMapSlice(
		oas.MediaTypes(content, oas.Join(response.Path, "content")),
		func(media oas.MediaType) []Finding {
			schema, path := c.resolve(oas.Get(media.Node, "schema"), oas.Join(media.Path, "schema"))
			if schema == nil {
				return nil
			}

			return c.container(schema, path)
		},
	)
}

// resolve follows $ref and tracks the component path of the target.
func (c errorContainerSchemaShapeChecker) resolve(node *yaml.Node, path string) (*yaml.Node, string) {
	return c.follow(node, path, errorContainerSchemaShapeMaxRefDepth)
}

// follow follows at most remaining references.
func (c errorContainerSchemaShapeChecker) follow(node *yaml.Node, path string, remaining int) (*yaml.Node, string) {
	ref := oas.Ref(node)
	target := oas.Pointer(c.root, ref)
	if remaining == 0 || ref == "" || target == nil {
		return node, path
	}

	return c.follow(target, errorContainerSchemaShapePointerPath(ref), remaining-1)
}

// errorContainerSchemaShapePointerPath converts #/a/b into $.a.b.
func errorContainerSchemaShapePointerPath(ref string) string {
	tokens := seq.Map(
		strings.SplitSeq(strings.TrimPrefix(strings.TrimPrefix(ref, "#"), "/"), "/"),
		func(token string) string {
			return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		},
	)

	return oas.JoinAll("$", tokens...)
}

// finding builds a finding.
func errorContainerSchemaShapeFinding(node *yaml.Node, path, message string) []Finding {
	return One(Finding{Node: node, Path: path, Message: message})
}

// withProperties reports whether schema is an object with a properties map.
func (c errorContainerSchemaShapeChecker) withProperties(schema *yaml.Node) bool {
	return oas.IsMap(schema) && oas.IsMap(oas.Get(schema, errorContainerSchemaShapeProperties))
}

// has reports whether the schema, or its composition, satisfies the predicate,
// like IBM's schemaHasConstraint: the schema itself, any allOf member, or every
// oneOf and anyOf member.
func (c errorContainerSchemaShapeChecker) has(candidate *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	schema := oas.Resolve(c.root, candidate)
	if !oas.IsMap(schema) || depth > errorContainerSchemaShapeMaxRefDepth {
		return false
	}

	if predicate(schema) {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, errorContainerSchemaShapeAllOf)) {
		if c.has(member, predicate, depth+1) {
			return true
		}
	}

	return slices.ContainsFunc([]string{literalOneof, errorContainerSchemaShapeAnyOf}, func(keyword string) bool {
		members := oas.Get(schema, keyword)

		return oas.IsSeq(members) && len(members.Content) > 0 &&
			!slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
				return !c.has(member, predicate, depth+1)
			})
	})
}

// ofType reports whether the schema is composition-aware of the given type.
func (c errorContainerSchemaShapeChecker) ofType(schema *yaml.Node, kind string) bool {
	return c.has(schema, func(node *yaml.Node) bool {
		return oas.HasType(node, kind) || (kind == errorContainerSchemaShapeArray && oas.Has(node, "items"))
	}, 0)
}

// property returns the resolved property schema and its path. When the property
// is missing the node is nil and the path is the properties object.
func (c errorContainerSchemaShapeChecker) property(properties *yaml.Node, path, name string) (*yaml.Node, string) {
	value := oas.Get(properties, name)
	if value == nil {
		return nil, path
	}

	return c.resolve(value, oas.Join(path, name))
}

// container checks the error container schema.
func (c errorContainerSchemaShapeChecker) container(schema *yaml.Node, path string) []Finding {
	if !c.withProperties(schema) {
		return errorContainerSchemaShapeFinding(schema, path, "Error container model must be an object with properties")
	}

	properties := oas.Get(schema, errorContainerSchemaShapeProperties)
	propertiesPath := oas.Join(path, errorContainerSchemaShapeProperties)

	return slices.Concat(
		c.typed(properties, propertiesPath, "trace",
			"Error container model should contain property 'trace' of type string"),
		c.statusCode(properties, propertiesPath),
		c.errors(properties, propertiesPath),
	)
}

// typed checks that a property exists and is a string schema.
func (c errorContainerSchemaShapeChecker) typed(properties *yaml.Node, path, name, message string) []Finding {
	property, propertyPath := c.property(properties, path, name)

	return When(!oas.IsMap(property) || !c.ofType(property, errorContainerSchemaShapeString), func() Finding {
		return Finding{Node: property, Path: propertyPath, Message: message}
	})
}

// statusCode checks the optional status_code property.
func (c errorContainerSchemaShapeChecker) statusCode(properties *yaml.Node, path string) []Finding {
	property, propertyPath := c.property(properties, path, "status_code")
	if property == nil && oas.Get(properties, "status_code") == nil {
		return nil
	}

	return When(!oas.IsMap(property) || !c.ofType(property, "integer"), func() Finding {
		return Finding{
			Node: property, Path: propertyPath,
			Message: "Error container model property 'status_code' must be of type integer",
		}
	})
}

// errorsItems returns the resolved items schema of the errors property.
func (c errorContainerSchemaShapeChecker) errorsItems(property *yaml.Node, path string) (*yaml.Node, string) {
	raw := oas.Get(property, "items")
	if raw == nil {
		return nil, path
	}

	return c.resolve(raw, oas.Join(path, "items"))
}

// errors checks the errors array of the container.
func (c errorContainerSchemaShapeChecker) errors(properties *yaml.Node, path string) []Finding {
	property, propertyPath := c.property(properties, path, "errors")
	if !oas.IsMap(property) || !c.ofType(property, errorContainerSchemaShapeArray) {
		return errorContainerSchemaShapeFinding(property, propertyPath,
			"Error container model must contain property 'errors' which must be an array of error models")
	}

	items, itemsPath := c.errorsItems(property, propertyPath)
	if !c.withProperties(items) {
		return errorContainerSchemaShapeFinding(property, itemsPath,
			"Error container model 'errors.items' field must be an object with properties")
	}

	return c.model(items, itemsPath)
}

// model checks one error model.
func (c errorContainerSchemaShapeChecker) model(schema *yaml.Node, path string) []Finding {
	properties := oas.Get(schema, errorContainerSchemaShapeProperties)
	propertiesPath := oas.Join(path, errorContainerSchemaShapeProperties)

	return slices.Concat(
		c.code(properties, propertiesPath),
		c.typed(properties, propertiesPath, "message",
			"Error model must contain property 'message' of type string"),
		c.typed(properties, propertiesPath, "more_info",
			"Error model should contain property 'more_info' of type string"),
		c.target(properties, propertiesPath),
	)
}

// code checks the code property and its enumeration.
func (c errorContainerSchemaShapeChecker) code(properties *yaml.Node, path string) []Finding {
	const typeMessage = "Error model must contain property 'code' of type string"

	code, codePath := c.property(properties, path, "code")
	if code == nil || !c.ofType(code, errorContainerSchemaShapeString) {
		return errorContainerSchemaShapeFinding(code, codePath, typeMessage)
	}

	const enumMessage = "Error model property 'code' must include an enumeration of the valid error codes"

	enum := oas.Get(code, errorContainerSchemaShapeEnum)
	switch {
	case enum == nil:
		return errorContainerSchemaShapeFinding(code, codePath, enumMessage)
	case !oas.IsSeq(enum):
		return errorContainerSchemaShapeFinding(enum, oas.Join(codePath, errorContainerSchemaShapeEnum), enumMessage)
	}

	return nil
}

// target checks the optional target of an error model.
func (c errorContainerSchemaShapeChecker) target(properties *yaml.Node, path string) []Finding {
	if oas.Get(properties, "target") == nil {
		return nil
	}

	target, targetPath := c.property(properties, path, "target")
	if !c.withProperties(target) {
		return errorContainerSchemaShapeFinding(target, targetPath,
			"Error model property 'target' must be a valid error target model object")
	}

	targetProperties := oas.Get(target, errorContainerSchemaShapeProperties)
	propertiesPath := oas.Join(targetPath, errorContainerSchemaShapeProperties)

	return slices.Concat(
		c.targetType(targetProperties, propertiesPath),
		c.typed(targetProperties, propertiesPath, "name",
			"Error target model must contain property 'name' of type string"),
	)
}

// targetType checks the type property and its enumeration.
func (c errorContainerSchemaShapeChecker) targetType(properties *yaml.Node, path string) []Finding {
	kind, kindPath := c.property(properties, path, "type")
	if kind == nil || !c.ofType(kind, errorContainerSchemaShapeString) {
		return errorContainerSchemaShapeFinding(
			kind,
			kindPath,
			"Error target model must contain property 'type' of type string",
		)
	}

	const message = "Error target model property 'type' must define an enumeration containing ['field', 'header', 'parameter']"

	enum := oas.Get(kind, errorContainerSchemaShapeEnum)
	switch {
	case enum == nil:
		return errorContainerSchemaShapeFinding(kind, kindPath, message)
	case !oas.IsSeq(enum) || !errorContainerSchemaShapeSameSet(oas.Strings(enum), len(enum.Content)):
		return errorContainerSchemaShapeFinding(enum, oas.Join(kindPath, errorContainerSchemaShapeEnum), message)
	}

	return nil
}

// errorContainerSchemaShapeSameSet reports whether the enumeration holds
// exactly field, parameter and header: the same length and every valid value
// present, in any order, as IBM's listsEqual does.
func errorContainerSchemaShapeSameSet(values []string, length int) bool {
	if length != len(errorContainerSchemaShapeTargetTypes()) {
		return false
	}

	return !slices.ContainsFunc(errorContainerSchemaShapeTargetTypes(), func(want string) bool {
		return !slices.Contains(values, want)
	})
}

// errorContainerSchemaShapeTargetTypes are the valid target types.
func errorContainerSchemaShapeTargetTypes() []string {
	return []string{"field", "parameter", "header"}
}

const (
	errorContainerSchemaShapeAllOf = "allOf"
	errorContainerSchemaShapeAnyOf = "anyOf"
)
