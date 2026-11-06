package rules

// Keywords and values that several rules compare against. They are shared so
// that a literal is spelled in exactly one place.
const (
	keywordInteger               = "integer"
	keywordHeader                = "header"
	keywordHeaders               = "headers"
	keywordList                  = "list"
	keywordMaxLength             = "maxLength"
	keywordMaxItems              = "maxItems"
	keywordAnyOf                 = "anyOf"
	keywordEnum                  = "enum"
	keywordForm                  = "form"
	keywordItems                 = "items"
	keywordExclusiveMaximum      = "exclusiveMaximum"
	keywordExclusiveMinimum      = "exclusiveMinimum"
	formatDate                   = "date"
	formatDateTime               = "date-time"
	formatInt32                  = "int32"
	formatInt64                  = "int64"
	keywordNumber                = "number"
	keywordUnevaluatedProperties = "unevaluatedProperties"
	keywordNot                   = "not"
	keywordNull                  = "null"
	keywordMaximum               = "maximum"
	keywordMinimum               = "minimum"
	keywordMinLength             = "minLength"
	keywordMaxProperties         = "maxProperties"
	keywordMinProperties         = "minProperties"
	keywordMinItems              = "minItems"
	formatOAS3                   = "oas3"
	formatOAS31                  = "oas3_1"
	keywordOAuth2                = "oauth2"
)

// keyValuePairWidth is the number of nodes a mapping uses for one entry.
const keyValuePairWidth = 2

// Further shared literals.
const (
	literalPath              = "path"
	literalOneof             = "oneOf"
	literalSchemas           = "schemas"
	literalRequestbodies     = "requestBodies"
	literalResponses         = "responses"
	literalParameters        = "parameters"
	literalQuery             = "query"
	literalRequired          = "required"
	literalString            = "string"
	literalTitle             = "title"
	literalPatternproperties = "patternProperties"
	literalPaths             = "paths"
	literalUUID              = "uuid"
	literalPattern           = "pattern"
	literalProperties        = "properties"
	literalSnake             = "snake"
	literalPascal            = "pascal"
	literalPatch             = "patch"
	literalObject            = "object"
)

// Further shared literals.
const (
	literalPost = "post"
)

// Further shared literals.
const (
	literalPut = "put"
)

// Lower-case HTTP method keywords as they appear in a path item. The values
// are deliberately not the upper-case net/http constants.
const (
	methodGet   = "get"
	methodPost  = "post"
	methodPut   = "put"
	methodPatch = "patch"
	methodHead  = "head"
)
