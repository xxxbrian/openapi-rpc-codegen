package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type EmitOptions struct {
	OutDir  string
	Check   bool
	Package string // default "server"
}

type ServerTemplateData struct {
	Package string
	BaseURL string

	Types    []GoTypeDecl
	Tags     []GoTag
	HasQuery bool
}

type GoTag struct {
	Name   string // sanitized Go ident
	Routes []GoRoute
}

type GoRoute struct {
	Name       string // operationId
	Method     string // GET/POST
	Path       string
	MethodName string // Get/Post

	TagName string // GoTag.Name

	PathType  string
	QueryType string
	BodyType  string
	RespType  string

	HasPath  bool
	HasQuery bool
	HasBody  bool

	// only generate local types when inline schema exists
	BodyInline bool
	RespInline bool

	// for path struct fields
	PathFields  []GoField
	QueryFields []GoQueryField

	HandlerName string // e.g. "handleGetUser"
}

type GoQueryField struct {
	Name      string
	JSONName  string
	Type      string
	Tag       string
	Required  bool
	ParseKind string // string|int64|float64|bool
	IsPointer bool
}

type GoTypeDecl struct {
	Name         string
	Kind         string // "struct" | "enum" | "alias"
	StructFields []GoField
	EnumValues   []string
	Alias        string
	Nullable     bool // for enums/aliases: we inline pointer logic; for struct: handled in field types
}

type GoField struct {
	Name     string // Exported Go field name
	JSONName string
	Type     string // Go type string
	Tag      string // struct tag, including omitempty if needed
}

func BuildServerData(spec *ir.Spec, pkg string) (*ServerTemplateData, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil IR spec")
	}
	if pkg == "" {
		pkg = "server"
	}

	data := &ServerTemplateData{
		Package: pkg,
		BaseURL: spec.Meta.BaseURL,
	}

	types, err := buildTypes(spec)
	if err != nil {
		return nil, err
	}
	data.Types = types

	tags, err := buildRoutes(spec)
	if err != nil {
		return nil, err
	}
	data.Tags = tags
	data.HasQuery = hasQuery(tags)

	return data, nil
}

func buildTypes(spec *ir.Spec) ([]GoTypeDecl, error) {
	names := make([]string, 0, len(spec.Types))
	for n := range spec.Types {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]GoTypeDecl, 0, len(names))
	for _, n := range names {
		td := spec.Types[n]
		goName := GoPublicIdent(td.Name)
		if goName == "" {
			return nil, fmt.Errorf("invalid type name: %q", td.Name)
		}

		switch td.Type.Kind {
		case ir.KindObject:
			fields := make([]GoField, 0, len(td.Type.Fields))
			for _, f := range td.Type.Fields {
				fieldName := GoPublicIdent(f.Name)
				if fieldName == "" {
					fieldName = "Field" + GoPublicIdent(n) // fallback
				}
				goType := renderGoTypeRef(f.Type, f.Required, td.Type.Nullable /*not used*/, false)
				tag := buildJSONTag(f.Name, f.Required)
				fields = append(fields, GoField{
					Name:     fieldName,
					JSONName: f.Name,
					Type:     goType,
					Tag:      tag,
				})
			}
			out = append(out, GoTypeDecl{
				Name:         goName,
				Kind:         "struct",
				StructFields: fields,
			})
		case ir.KindEnum:
			out = append(out, GoTypeDecl{
				Name:       goName,
				Kind:       "enum",
				EnumValues: td.Type.Enum,
			})
		case ir.KindScalar, ir.KindArray:
			out = append(out, GoTypeDecl{
				Name:  goName,
				Kind:  "alias",
				Alias: renderGoInlineType(td.Type),
			})
		default:
			return nil, fmt.Errorf("unsupported type kind: %s", td.Type.Kind)
		}
	}
	return out, nil
}

func buildRoutes(spec *ir.Spec) ([]GoTag, error) {
	byTag := map[string][]ir.Route{}
	tagSet := map[string]bool{}

	for _, r := range spec.Routes {
		tag := GoPublicIdent(r.Tag)
		if tag == "" {
			tag = "Default"
		}
		byTag[tag] = append(byTag[tag], r)
		tagSet[tag] = true
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	out := make([]GoTag, 0, len(tags))
	for _, t := range tags {
		rs := byTag[t]
		sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })

		gt := GoTag{Name: t}
		for _, r := range rs {
			gr, err := toGoRoute(t, r, spec.Types)
			if err != nil {
				return nil, err
			}
			gt.Routes = append(gt.Routes, gr)
		}
		out = append(out, gt)
	}
	return out, nil
}

func toGoRoute(tag string, r ir.Route, types map[string]ir.TypeDecl) (GoRoute, error) {
	if r.Success.Type == nil {
		return GoRoute{}, fmt.Errorf("%s: missing 200 response schema", r.Name)
	}

	op := GoPublicIdent(r.Name)
	if op == "" {
		return GoRoute{}, fmt.Errorf("invalid operationId: %q", r.Name)
	}

	methodName := methodName(r.Method)
	if methodName == "" {
		return GoRoute{}, fmt.Errorf("%s: unsupported method %q", r.Name, r.Method)
	}

	hasPath := len(r.PathParams) > 0
	hasQuery := len(r.QueryParams) > 0
	hasBody := r.Method == "POST" && r.RequestBody != nil

	pathType := ""
	queryType := ""
	if hasPath {
		pathType = op + "Path"
	}
	if hasQuery {
		queryType = op + "Query"
	}

	// Prefer global types when $ref exists.
	bodyType := ""
	bodyInline := false
	if hasBody {
		bodyInline = (r.RequestBody.Type.RefName == "")
		bodyType = goTypeFromTypeRef(r.RequestBody.Type, op+"Body")
	}

	respInline := (r.Success.Type.RefName == "")
	respType := goTypeFromTypeRef(*r.Success.Type, op+"Result") // avoid "Response" collisions

	// Build path fields (string-only for now; can later type via IR)
	var pathFields []GoField
	if hasPath {
		for _, p := range r.PathParams {
			fn := GoPublicIdent(p.Name)
			if fn == "" {
				return GoRoute{}, fmt.Errorf("%s: invalid path param name %q", r.Name, p.Name)
			}
			pathFields = append(pathFields, GoField{
				Name:     fn,
				JSONName: p.Name,
				Type:     "string",
				Tag:      "",
			})
		}
	}

	var queryFields []GoQueryField
	if hasQuery {
		for _, p := range r.QueryParams {
			fn := GoPublicIdent(p.Name)
			if fn == "" {
				return GoRoute{}, fmt.Errorf("%s: invalid query param name %q", r.Name, p.Name)
			}
			scalarKind, ok := scalarKindForTypeRef(p.Type, types)
			if !ok {
				return GoRoute{}, fmt.Errorf("%s: unsupported query param type %q", r.Name, p.Name)
			}
			parseKind := parseKindForScalar(scalarKind)
			goType := renderGoTypeRef(p.Type, p.Required, false, false)
			queryFields = append(queryFields, GoQueryField{
				Name:      fn,
				JSONName:  p.Name,
				Type:      goType,
				Tag:       buildJSONTag(p.Name, p.Required),
				Required:  p.Required,
				ParseKind: parseKind,
				IsPointer: strings.HasPrefix(goType, "*"),
			})
		}
	}

	return GoRoute{
		Name:       r.Name,
		Method:     r.Method,
		Path:       r.Path,
		MethodName: methodName,

		TagName: tag,

		PathType:  pathType,
		QueryType: queryType,
		BodyType:  bodyType,
		RespType:  respType,

		HasPath:  hasPath,
		HasQuery: hasQuery,
		HasBody:  hasBody,

		BodyInline: bodyInline,
		RespInline: respInline,

		PathFields:  pathFields,
		QueryFields: queryFields,

		HandlerName: "handle" + op,
	}, nil
}

func goTypeFromTypeRef(tr ir.TypeRef, fallback string) string {
	if tr.RefName != "" {
		return GoPublicIdent(tr.RefName)
	}
	if tr.Inline != nil {
		return fallback
	}
	return "any"
}

func methodName(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	default:
		return ""
	}
}

func scalarKindForTypeRef(tr ir.TypeRef, types map[string]ir.TypeDecl) (string, bool) {
	if tr.Inline != nil {
		switch tr.Inline.Kind {
		case ir.KindScalar:
			return tr.Inline.Scalar, true
		case ir.KindEnum:
			return "string", true
		default:
			return "", false
		}
	}
	if tr.RefName != "" {
		td, ok := types[tr.RefName]
		if !ok {
			return "", false
		}
		switch td.Type.Kind {
		case ir.KindScalar:
			return td.Type.Scalar, true
		case ir.KindEnum:
			return "string", true
		default:
			return "", false
		}
	}
	return "", false
}

func parseKindForScalar(kind string) string {
	switch kind {
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

func hasQuery(tags []GoTag) bool {
	for _, tag := range tags {
		for _, route := range tag.Routes {
			if route.HasQuery {
				return true
			}
		}
	}
	return false
}

// ---- helpers ----

func buildJSONTag(jsonName string, required bool) string {
	if required {
		return fmt.Sprintf("`json:%q`", jsonName)
	}
	return fmt.Sprintf("`json:%q`", jsonName+",omitempty")
}

func GoPublicIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Split by non-alnum, Title-case each part
	var parts []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		parts = append(parts, cur.String())
		cur.Reset()
	}
	for _, r := range s {
		isLetter := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		isDigit := (r >= '0' && r <= '9')
		if isLetter || isDigit {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	if len(parts) == 0 {
		return ""
	}
	var out strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		out.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			out.WriteString(p[1:])
		}
	}
	res := out.String()
	// Must start with letter
	if res == "" || (res[0] >= '0' && res[0] <= '9') {
		return ""
	}
	return res
}

func renderGoTypeRef(tr ir.TypeRef, required bool, _ bool, _ bool) string {
	// required=false => pointer for scalar/struct/alias to express "optional"
	// nullable=true already encoded in inline types as pointers in renderGoInlineType.
	if tr.RefName != "" {
		t := GoPublicIdent(tr.RefName)
		if !required {
			return "*" + t
		}
		return t
	}
	if tr.Inline != nil {
		t := renderGoInlineType(*tr.Inline)
		if !required && !strings.HasPrefix(t, "*") && !strings.HasPrefix(t, "[]") {
			return "*" + t
		}
		return t
	}
	return "any"
}

func renderGoInlineType(t ir.Type) string {
	switch t.Kind {
	case ir.KindScalar:
		base := "any"
		switch t.Scalar {
		case "string":
			base = "string"
		case "number":
			base = "float64"
		case "integer":
			base = "int64"
		case "boolean":
			base = "bool"
		}
		if t.Nullable {
			return "*" + base
		}
		return base
	case ir.KindEnum:
		// inline enum: represent as string (simplify)
		if t.Nullable {
			return "*string"
		}
		return "string"
	case ir.KindArray:
		elem := "any"
		if t.Elem != nil {
			elem = renderGoTypeRef(*t.Elem, true, false, false)
			// if elem is pointer, slice of pointers is ok
		}
		base := "[]" + elem
		if t.Nullable {
			// nullable array => pointer to slice
			return "*" + base
		}
		return base
	case ir.KindObject:
		// inline object -> inline struct
		var b strings.Builder
		b.WriteString("struct {\n")
		for _, f := range t.Fields {
			fn := GoPublicIdent(f.Name)
			if fn == "" {
				fn = "Field"
			}
			ft := renderGoTypeRef(f.Type, f.Required, false, false)
			tag := buildJSONTag(f.Name, f.Required)
			b.WriteString("  ")
			b.WriteString(fn)
			b.WriteString(" ")
			b.WriteString(ft)
			b.WriteString(" ")
			b.WriteString(tag)
			b.WriteString("\n")
		}
		b.WriteString("}")
		if t.Nullable {
			return "*" + b.String()
		}
		return b.String()
	default:
		return "any"
	}
}
