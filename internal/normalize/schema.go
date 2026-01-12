package normalize

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

// SchemaRefToTypeRef converts OpenAPI schema to our IR TypeRef.
// disallow oneOf/anyOf/allOf/additionalProperties entirely.
func SchemaRefToTypeRef(sr *openapi3.SchemaRef) (ir.TypeRef, error) {
	if sr == nil {
		return ir.TypeRef{}, fmt.Errorf("schema is nil")
	}

	// $ref: only allow #/components/schemas/*
	if sr.Ref != "" {
		name, ok := refToComponentName(sr.Ref)
		if !ok {
			return ir.TypeRef{}, fmt.Errorf("only $ref to #/components/schemas/* is supported; got %q", sr.Ref)
		}
		return ir.TypeRef{RefName: name}, nil
	}

	if sr.Value == nil {
		return ir.TypeRef{}, fmt.Errorf("schema has no value")
	}

	t, err := schemaValueToType(sr.Value)
	if err != nil {
		return ir.TypeRef{}, err
	}
	return ir.TypeRef{Inline: &t}, nil
}

func schemaValueToType(s *openapi3.Schema) (ir.Type, error) {
	if s == nil {
		return ir.Type{}, fmt.Errorf("schema is nil")
	}

	// Forbidden combinators & dynamic maps
	if len(s.OneOf) > 0 {
		return ir.Type{}, fmt.Errorf("oneOf is not supported")
	}
	if len(s.AnyOf) > 0 {
		return ir.Type{}, fmt.Errorf("anyOf is not supported")
	}
	if len(s.AllOf) > 0 {
		return ir.Type{}, fmt.Errorf("allOf is not supported")
	}
	if s.AdditionalProperties.Has != nil {
		// additionalProperties can be boolean or schema; both disallowed
		return ir.Type{}, fmt.Errorf("additionalProperties is not supported")
	}

	out := ir.Type{
		Nullable: s.Nullable,
	}

	// Enum (support string enums only)
	if len(s.Enum) > 0 {
		vals := make([]string, 0, len(s.Enum))
		for _, v := range s.Enum {
			str, ok := v.(string)
			if !ok {
				return ir.Type{}, fmt.Errorf("enum must be string values only")
			}
			vals = append(vals, str)
		}
		out.Kind = ir.KindEnum
		out.Enum = vals
		return out, nil
	}

	typ, err := primaryType(s)
	if err != nil {
		return ir.Type{}, err
	}

	switch typ {
	case "string":
		out.Kind = ir.KindScalar
		out.Scalar = "string"
		return out, nil
	case "number":
		out.Kind = ir.KindScalar
		out.Scalar = "number"
		return out, nil
	case "integer":
		out.Kind = ir.KindScalar
		out.Scalar = "integer"
		return out, nil
	case "boolean":
		out.Kind = ir.KindScalar
		out.Scalar = "boolean"
		return out, nil
	case "array":
		if s.Items == nil {
			return ir.Type{}, fmt.Errorf("array must define items")
		}
		elem, err := SchemaRefToTypeRef(s.Items)
		if err != nil {
			return ir.Type{}, fmt.Errorf("array items: %w", err)
		}
		out.Kind = ir.KindArray
		out.Elem = &elem
		return out, nil
	case "object":
		// object: only properties/required allowed
		fields := make([]ir.Field, 0, len(s.Properties))
		required := make(map[string]bool, len(s.Required))
		for _, n := range s.Required {
			required[n] = true
		}

		names := make([]string, 0, len(s.Properties))
		for k := range s.Properties {
			names = append(names, k)
		}
		sort.Strings(names)

		for _, name := range names {
			prop := s.Properties[name]
			if prop == nil {
				return ir.Type{}, fmt.Errorf("property %q schema is nil", name)
			}
			tr, err := SchemaRefToTypeRef(prop)
			if err != nil {
				return ir.Type{}, fmt.Errorf("property %q: %w", name, err)
			}
			fields = append(fields, ir.Field{
				Name:     name,
				Required: required[name],
				Type:     tr,
			})
		}

		out.Kind = ir.KindObject
		out.Fields = fields
		return out, nil
	default:
		return ir.Type{}, fmt.Errorf("schema type %q is not supported (must be object/array/string/number/integer/boolean or enum)", typ)
	}
}

// primaryType extracts a single effective type from kin-openapi's *openapi3.Types.
// keep it strict: if multiple types are present, reject.
func primaryType(s *openapi3.Schema) (string, error) {
	if s.Type == nil || len(*s.Type) == 0 {
		// We stay strict to avoid “implicit object” ambiguity.
		return "", fmt.Errorf("schema must declare type (no implicit typing)")
	}
	types := *s.Type
	if len(types) != 1 {
		return "", fmt.Errorf("multiple schema types are not supported: %v", types)
	}
	t := strings.TrimSpace(types[0])
	if t == "" {
		return "", fmt.Errorf("schema type is empty")
	}
	return t, nil
}

func refToComponentName(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return "", false
	}
	return name, true
}
