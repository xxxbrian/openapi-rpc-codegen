package normalize

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

func normalizeSuccessResponse(op *openapi3.Operation) (ir.Success, error) {
	if op.Responses == nil {
		return ir.Success{}, fmt.Errorf("responses is missing")
	}

	// Strict: only "200" is allowed
	keys := make([]string, 0, len(op.Responses.Map()))
	for k := range op.Responses.Map() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		return ir.Success{}, fmt.Errorf("responses is empty; must define 200")
	}
	if len(keys) != 1 || keys[0] != "200" {
		return ir.Success{}, fmt.Errorf("only responses[\"200\"] is allowed; found: %v", keys)
	}

	r200 := op.Responses.Map()["200"]
	if r200 == nil || r200.Value == nil {
		return ir.Success{}, fmt.Errorf("responses[\"200\"] is nil")
	}

	mt := "application/json"
	content := r200.Value.Content
	if content == nil || content[mt] == nil {
		// also allow "application/json; charset=utf-8"? (some specs do)
		alt, schema := findJSONContent(content)
		if schema == nil {
			return ir.Success{}, fmt.Errorf("responses[\"200\"] must have %q content", mt)
		}
		typ, err := SchemaRefToTypeRef(schema)
		if err != nil {
			return ir.Success{}, fmt.Errorf("responses[\"200\"] %q schema: %w", alt, err)
		}
		return ir.Success{Status: "200", Type: &typ}, nil
	}

	if content[mt].Schema == nil {
		return ir.Success{}, fmt.Errorf("responses[\"200\"] %q must define schema", mt)
	}

	typ, err := SchemaRefToTypeRef(content[mt].Schema)
	if err != nil {
		return ir.Success{}, fmt.Errorf("responses[\"200\"] %q schema: %w", mt, err)
	}

	return ir.Success{
		Status: "200",
		Type:   &typ,
	}, nil
}

// findJSONContent tries to locate a content key that is effectively JSON.
// returns (contentTypeKey, schemaRef)
func findJSONContent(content openapi3.Content) (string, *openapi3.SchemaRef) {
	if content == nil {
		return "", nil
	}
	// accept "application/json; charset=utf-8" etc.
	for k, v := range content {
		kk := strings.ToLower(strings.TrimSpace(k))
		if strings.HasPrefix(kk, "application/json") && v != nil && v.Schema != nil {
			return k, v.Schema
		}
	}
	return "", nil
}
