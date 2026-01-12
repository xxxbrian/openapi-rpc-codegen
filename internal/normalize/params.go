package normalize

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

func collectParams(item *openapi3.PathItem, op *openapi3.Operation) ([]ir.Param, []ir.Param, error) {
	merged := make([]*openapi3.ParameterRef, 0, len(item.Parameters)+len(op.Parameters))
	merged = append(merged, item.Parameters...)
	merged = append(merged, op.Parameters...)

	seen := map[string]struct{}{} // in:name

	var pathParams []ir.Param
	var queryParams []ir.Param

	for _, pr := range merged {
		if pr == nil || pr.Value == nil {
			return nil, nil, fmt.Errorf("parameter is nil")
		}
		p := pr.Value

		in := strings.TrimSpace(p.In)
		name := strings.TrimSpace(p.Name)
		if in == "" || name == "" {
			return nil, nil, fmt.Errorf("parameter has empty in/name")
		}

		// strict: only path/query
		if in != "path" && in != "query" {
			return nil, nil, fmt.Errorf("parameter %q in %q is not supported (only path/query)", name, in)
		}

		key := in + ":" + name
		if _, ok := seen[key]; ok {
			// ignore duplicates deterministically (PathItem + Operation)
			continue
		}
		seen[key] = struct{}{}

		// Path param must be required
		required := p.Required
		if in == "path" {
			required = true
		}

		if p.Schema == nil {
			return nil, nil, fmt.Errorf("parameter %q in %q must define schema", name, in)
		}

		typ, err := SchemaRefToTypeRef(p.Schema)
		if err != nil {
			return nil, nil, fmt.Errorf("parameter %q in %q: %w", name, in, err)
		}

		param := ir.Param{
			Name:     name,
			Required: required,
			Type:     typ,
		}

		if in == "path" {
			pathParams = append(pathParams, param)
		} else {
			queryParams = append(queryParams, param)
		}
	}

	// stable order
	sort.Slice(pathParams, func(i, j int) bool { return pathParams[i].Name < pathParams[j].Name })
	sort.Slice(queryParams, func(i, j int) bool { return queryParams[i].Name < queryParams[j].Name })

	return pathParams, queryParams, nil
}
