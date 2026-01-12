package normalize

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	BaseURLOverride string
}

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ToIR(doc *openapi3.T, opt Options) (*ir.Spec, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil OpenAPI doc")
	}

	baseURL := strings.TrimSpace(doc.Servers[0].URL)
	if opt.BaseURLOverride != "" {
		baseURL = strings.TrimSpace(opt.BaseURLOverride)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is empty (servers[0].url or --base-url)")
	}

	out := &ir.Spec{
		Meta:   ir.Meta{BaseURL: baseURL},
		Types:  map[string]ir.TypeDecl{}, // filled later (optional)
		Routes: []ir.Route{},
	}

	// Enforce: every operation must have unique operationId
	seenOpID := map[string]string{} // opId -> "METHOD path"

	// deterministic traversal
	paths := sortedKeys(doc.Paths.Map())
	for _, p := range paths {
		item := doc.Paths.Map()[p]
		if item == nil {
			continue
		}

		// Only support GET/POST
		for _, m := range []string{"GET", "POST"} {
			op := operationByMethod(item, m)
			if op == nil {
				continue
			}

			loc := fmt.Sprintf("%s %s", m, p)

			// operationId must exist and be a legal identifier
			opID := strings.TrimSpace(op.OperationID)
			if opID == "" {
				return nil, fmt.Errorf("%s: missing operationId (required)", loc)
			}
			if !identRe.MatchString(opID) {
				return nil, fmt.Errorf("%s: invalid operationId %q (must match %s)", loc, opID, identRe.String())
			}
			if prev, ok := seenOpID[opID]; ok {
				return nil, fmt.Errorf("%s: duplicate operationId %q (already used by %s)", loc, opID, prev)
			}
			seenOpID[opID] = loc

			// tags[0] for grouping
			tag := "Default"
			if len(op.Tags) > 0 && strings.TrimSpace(op.Tags[0]) != "" {
				tag = sanitizeIdent(op.Tags[0])
				if tag == "" {
					tag = "Default"
				}
			}

			// GET must not have requestBody
			if m == "GET" && op.RequestBody != nil {
				return nil, fmt.Errorf("%s: requestBody is not allowed for GET", loc)
			}

			// parameters (path/query only)
			pathParams, queryParams, err := collectParams(item, op)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", loc, err)
			}

			// request body (POST only; optional)
			var reqBody *ir.Body
			if m == "POST" {
				reqBody, err = normalizeRequestBody(op)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", loc, err)
				}
			}

			// responses: must be only 200 + application/json + schema
			success, err := normalizeSuccessResponse(op)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", loc, err)
			}

			out.Routes = append(out.Routes, ir.Route{
				Name:        opID,
				Tag:         tag,
				Method:      m,
				Path:        p,
				PathParams:  pathParams,
				QueryParams: queryParams,
				RequestBody: reqBody,
				Success:     success,
			})
		}

		// Reject other HTTP methods if present (strict)
		if hasUnsupportedMethods(item) {
			return nil, fmt.Errorf("%s: contains unsupported HTTP method (only GET/POST allowed)", p)
		}
	}

	// stable order: by Tag then Name
	sort.Slice(out.Routes, func(i, j int) bool {
		if out.Routes[i].Tag != out.Routes[j].Tag {
			return out.Routes[i].Tag < out.Routes[j].Tag
		}
		return out.Routes[i].Name < out.Routes[j].Name
	})

	return out, nil
}

func operationByMethod(item *openapi3.PathItem, method string) *openapi3.Operation {
	switch method {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	default:
		return nil
	}
}

func hasUnsupportedMethods(item *openapi3.PathItem) bool {
	// Any method outside GET/POST triggers strict error if defined
	if item.Put != nil || item.Delete != nil || item.Patch != nil || item.Options != nil || item.Head != nil || item.Trace != nil {
		return true
	}
	return false
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sanitizeIdent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// keep letters/numbers/_ ; replace others with _
	var b strings.Builder
	for i, r := range s {
		isLetter := (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		isDigit := (r >= '0' && r <= '9')
		if isLetter || isDigit || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
		if i == 0 {
			// if first is digit, prefix underscore
			// (we handle after build too, but early is fine)
		}
	}
	out := b.String()
	if out == "" {
		return ""
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	// collapse multiple underscores
	out = strings.Join(strings.FieldsFunc(out, func(r rune) bool { return r == '_' }), "_")
	if out == "" {
		return ""
	}
	return out
}
