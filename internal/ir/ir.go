package ir

type Spec struct {
	Meta   Meta
	Types  map[string]TypeDecl
	Routes []Route
}

type Meta struct {
	BaseURL string
}

type Route struct {
	Name   string
	Tag    string
	Method string
	Path   string

	PathParams  []Param
	QueryParams []Param

	RequestBody *Body
	Success     Success
}

type Param struct {
	Name     string
	Required bool
	Type     TypeRef
}

type Body struct {
	Required bool
	Type     TypeRef
}

type Success struct {
	Status string
	Type   *TypeRef
}

// ---- Types ----

type TypeDecl struct {
	Name string
	Type Type
}

type TypeRef struct {
	RefName string
	Inline  *Type
}

type TypeKind string

const (
	KindScalar TypeKind = "scalar"
	KindObject TypeKind = "object"
	KindArray  TypeKind = "array"
	KindEnum   TypeKind = "enum"
)

type Type struct {
	Kind TypeKind

	// scalar
	Scalar string // "string" | "number" | "integer" | "boolean"

	// object
	Fields []Field

	// array
	Elem *TypeRef

	// enum
	Enum []string

	// nullable (OpenAPI 3)
	Nullable bool
}

type Field struct {
	Name     string
	Required bool
	Type     TypeRef
}
