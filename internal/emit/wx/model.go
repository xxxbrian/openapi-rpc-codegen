package wx

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type EmitOptions struct {
	OutDir string
	Check  bool
}

type TypesTemplateData struct {
	Types []NamedType
}

type NamedType struct {
	Name string
	Kind string // "object" | "enum" | "alias"
	// For object
	Fields []Field
	// For enum
	Enum []string
	// For alias (scalar/array/inline object)
	Alias string

	// Nullable: for non-object kinds we emit `| null` inline.
	// For object, we may emit `export type XNullable = X | null;`
	Nullable bool
}

type Field struct {
	Name     string
	Optional bool
	Type     string // TS type string
}

type ClientTemplateData struct {
	Tags []ClientTag
}

type ClientTag struct {
	Name   string // sanitized identifier
	Routes []ClientRoute
}

type ClientRoute struct {
	Name       string
	Method     string // GET/POST
	PathExpr   string // template literal or quoted string
	Signature  string // e.g. "path: {...}, query?: {...}"
	ReturnType string // e.g. "T.User"
	BodyVar    string // "body" or "undefined"
	QueryVar   string // "query" or "undefined"
}

func BuildTypesData(spec *ir.Spec) (*TypesTemplateData, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil IR spec")
	}
	names := make([]string, 0, len(spec.Types))
	for n := range spec.Types {
		names = append(names, n)
	}
	sort.Strings(names)

	out := &TypesTemplateData{Types: make([]NamedType, 0, len(names))}
	for _, name := range names {
		td := spec.Types[name]
		nt, err := toNamedType(td)
		if err != nil {
			return nil, fmt.Errorf("type %s: %w", name, err)
		}
		out.Types = append(out.Types, nt)
	}
	return out, nil
}

func toNamedType(td ir.TypeDecl) (NamedType, error) {
	name := sanitizeTSIdent(td.Name)
	if name == "" {
		return NamedType{}, fmt.Errorf("invalid type name %q", td.Name)
	}

	switch td.Type.Kind {
	case ir.KindObject:
		nt := NamedType{Name: name, Kind: "object", Nullable: td.Type.Nullable}
		for _, f := range td.Type.Fields {
			nt.Fields = append(nt.Fields, Field{
				Name:     f.Name,
				Optional: !f.Required,
				Type:     renderTypeRefAsTS(f.Type),
			})
		}
		return nt, nil
	case ir.KindEnum:
		return NamedType{
			Name:     name,
			Kind:     "enum",
			Enum:     td.Type.Enum,
			Nullable: td.Type.Nullable,
		}, nil
	case ir.KindScalar, ir.KindArray:
		return NamedType{
			Name:     name,
			Kind:     "alias",
			Alias:    renderInlineTypeAsTS(td.Type),
			Nullable: false, // already included by renderInlineTypeAsTS if needed
		}, nil
	default:
		return NamedType{}, fmt.Errorf("unsupported kind: %s", td.Type.Kind)
	}
}

func BuildClientData(spec *ir.Spec) (*ClientTemplateData, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil IR spec")
	}

	byTag := map[string][]ir.Route{}
	tagSet := map[string]bool{}
	for _, r := range spec.Routes {
		tag := sanitizeTSIdent(r.Tag)
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

	data := &ClientTemplateData{Tags: make([]ClientTag, 0, len(tags))}
	for _, t := range tags {
		rs := byTag[t]
		sort.Slice(rs, func(i, j int) bool { return rs[i].Name < rs[j].Name })

		ct := ClientTag{Name: t}
		for _, r := range rs {
			cr, err := toClientRoute(r)
			if err != nil {
				return nil, fmt.Errorf("route %s.%s: %w", t, r.Name, err)
			}
			ct.Routes = append(ct.Routes, cr)
		}
		data.Tags = append(data.Tags, ct)
	}

	return data, nil
}

func toClientRoute(r ir.Route) (ClientRoute, error) {
	if r.Success.Type == nil {
		return ClientRoute{}, fmt.Errorf("success schema missing (Scheme A requires 200 JSON schema)")
	}

	ret := "T." + sanitizeTSIdent(renderTypeRefName(*r.Success.Type))
	if ret == "T." {
		ret = "unknown"
	}

	// signature:
	// POST with body: (body: X, path?: {...}, query?: {...})
	// GET: (path: {...}, query?: {...}) or (query?: {...}) etc.
	args := []string{}
	if r.Method == "POST" && r.RequestBody != nil {
		args = append(args, "body: "+renderTypeRefAsTS(r.RequestBody.Type))
	}
	if len(r.PathParams) > 0 {
		args = append(args, "path: "+renderParamsObjType(r.PathParams))
	}
	if len(r.QueryParams) > 0 {
		args = append(args, "query?: "+renderParamsObjType(r.QueryParams))
	}
	sig := strings.Join(args, ", ")

	bodyVar := "undefined"
	if r.Method == "POST" && r.RequestBody != nil {
		bodyVar = "body"
	}
	queryVar := "undefined"
	if len(r.QueryParams) > 0 {
		queryVar = "query"
	}

	return ClientRoute{
		Name:       r.Name,
		Method:     r.Method,
		PathExpr:   renderPathExpr(r.Path, r.PathParams),
		Signature:  sig,
		ReturnType: ret,
		BodyVar:    bodyVar,
		QueryVar:   queryVar,
	}, nil
}

func renderParamsObjType(ps []ir.Param) string {
	var b strings.Builder
	b.WriteString("{ ")
	for i, p := range ps {
		if i > 0 {
			b.WriteString("; ")
		}
		prop := p.Name
		if !isSafeTSProp(prop) {
			prop = fmt.Sprintf("%q", prop)
		}
		b.WriteString(prop)
		if !p.Required {
			b.WriteString("?")
		}
		b.WriteString(": ")
		b.WriteString(renderTypeRefAsTS(p.Type))
	}
	b.WriteString(" }")
	return b.String()
}

func renderPathExpr(path string, params []ir.Param) string {
	if len(params) == 0 {
		return fmt.Sprintf("%q", path)
	}
	out := path
	for _, p := range params {
		needle := "{" + p.Name + "}"
		repl := "${encodeURIComponent(String(path" + tsAccess(p.Name) + "))}"
		out = strings.ReplaceAll(out, needle, repl)
	}
	return "`" + out + "`"
}

func tsAccess(name string) string {
	if isSafeTSProp(name) {
		return "." + name
	}
	return "[" + fmt.Sprintf("%q", name) + "]"
}

func renderTypeRefName(tr ir.TypeRef) string {
	if tr.RefName != "" {
		return tr.RefName
	}
	// inline types don't have names; for return type in Scheme A we expect refs most of the time
	return "unknown"
}

// --- TS type rendering (shared with both emitters) ---

func renderTypeRefAsTS(tr ir.TypeRef) string {
	if tr.RefName != "" {
		return "T." + sanitizeTSIdent(tr.RefName)
	}
	if tr.Inline != nil {
		return renderInlineTypeAsTS(*tr.Inline)
	}
	return "unknown"
}

func renderInlineTypeAsTS(t ir.Type) string {
	switch t.Kind {
	case ir.KindScalar:
		switch t.Scalar {
		case "string":
			return withNull("string", t.Nullable)
		case "number", "integer":
			return withNull("number", t.Nullable)
		case "boolean":
			return withNull("boolean", t.Nullable)
		default:
			return withNull("unknown", t.Nullable)
		}
	case ir.KindEnum:
		u := strings.Join(quoteUnion(t.Enum), " | ")
		return withNull(u, t.Nullable)
	case ir.KindArray:
		if t.Elem == nil {
			return withNull("unknown[]", t.Nullable)
		}
		return withNull(renderTypeRefAsTS(*t.Elem)+"[]", t.Nullable)
	case ir.KindObject:
		// inline object literal
		var b strings.Builder
		b.WriteString("{ ")
		for i, f := range t.Fields {
			if i > 0 {
				b.WriteString("; ")
			}
			prop := f.Name
			if !isSafeTSProp(prop) {
				prop = fmt.Sprintf("%q", prop)
			}
			b.WriteString(prop)
			if !f.Required {
				b.WriteString("?")
			}
			b.WriteString(": ")
			b.WriteString(renderTypeRefAsTS(f.Type))
		}
		b.WriteString(" }")
		return withNull(b.String(), t.Nullable)
	default:
		return withNull("unknown", t.Nullable)
	}
}

func withNull(s string, nullable bool) string {
	if nullable {
		return s + " | null"
	}
	return s
}

func quoteUnion(vals []string) []string {
	if len(vals) == 0 {
		return []string{"never"}
	}
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, fmt.Sprintf("%q", v))
	}
	return out
}

func sanitizeTSIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		isLetter := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		isDigit := (r >= '0' && r <= '9')
		if isLetter || isDigit || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out != "" && out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	return out
}

func isSafeTSProp(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		isLetter := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || r == '$'
		isDigit := (r >= '0' && r <= '9')
		if i == 0 {
			if !isLetter {
				return false
			}
		} else {
			if !(isLetter || isDigit) {
				return false
			}
		}
	}
	return true
}
