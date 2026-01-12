package wx

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit/common"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

//go:embed templates/types.ts.tpl
var typesTplFS embed.FS

func EmitTypes(spec *ir.Spec, opt EmitOptions) ([]string, error) {
	data, err := BuildTypesData(spec)
	if err != nil {
		return nil, err
	}

	tplText, err := typesTplFS.ReadFile("templates/types.ts.tpl")
	if err != nil {
		return nil, fmt.Errorf("read types template: %w", err)
	}

	funcs := template.FuncMap{
		"enumUnion": func(vals []string) string {
			qs := quoteUnion(vals)
			// qs already quoted
			out := qs[0]
			for i := 1; i < len(qs); i++ {
				out += " | " + qs[i]
			}
			return out
		},
		"isSafeProp": isSafeTSProp,
	}

	tpl, err := template.New("types").Funcs(funcs).Parse(string(tplText))
	if err != nil {
		return nil, fmt.Errorf("parse types template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("exec types template: %w", err)
	}

	outPath := filepath.Join(opt.OutDir, "types.gen.ts")
	wrote, err := common.WriteFile(outPath, buf.Bytes(), common.WriteOptions{Check: opt.Check})
	if err != nil {
		return nil, err
	}
	if wrote {
		return []string{outPath}, nil
	}
	return []string{}, nil
}
