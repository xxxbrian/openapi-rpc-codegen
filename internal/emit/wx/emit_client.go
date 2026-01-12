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

//go:embed templates/client.ts.tpl
var clientTplFS embed.FS

func EmitClient(spec *ir.Spec, opt EmitOptions) ([]string, error) {
	data, err := BuildClientData(spec)
	if err != nil {
		return nil, err
	}

	tplText, err := clientTplFS.ReadFile("templates/client.ts.tpl")
	if err != nil {
		return nil, fmt.Errorf("read client template: %w", err)
	}

	tpl, err := template.New("client").Parse(string(tplText))
	if err != nil {
		return nil, fmt.Errorf("parse client template: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("exec client template: %w", err)
	}

	outPath := filepath.Join(opt.OutDir, "client.gen.ts")
	wrote, err := common.WriteFile(outPath, buf.Bytes(), common.WriteOptions{Check: opt.Check})
	if err != nil {
		return nil, err
	}
	if wrote {
		return []string{outPath}, nil
	}
	return []string{}, nil
}
