package emit

import (
	"fmt"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit/go/server"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit/ts/wx"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	OutDir  string
	Targets []string
	Check   bool
	Verbose bool
}

func Dispatch(spec *ir.Spec, opt Options) ([]string, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil IR spec")
	}
	if len(opt.Targets) == 0 {
		// default target
		opt.Targets = []string{"raw-ir"}
	}
	var files []string
	for _, t := range opt.Targets {
		switch t {
		case "raw-ir":
			// TODO: Emit raw IR for debugging
			fmt.Printf("%+v\n", spec)
		case "ts-wx":
			fs1, err := wx.EmitTypes(spec, wx.EmitOptions{OutDir: opt.OutDir, Check: opt.Check})
			if err != nil {
				return nil, err
			}
			files = append(files, fs1...)

			fs2, err := wx.EmitTransport(spec, wx.EmitOptions{OutDir: opt.OutDir, Check: opt.Check})
			if err != nil {
				return nil, err
			}
			files = append(files, fs2...)

			fs3, err := wx.EmitClient(spec, wx.EmitOptions{OutDir: opt.OutDir, Check: opt.Check})
			if err != nil {
				return nil, err
			}
			files = append(files, fs3...)
		case "go-server":
			fs, err := server.Emit(spec, server.EmitOptions{
				OutDir:  opt.OutDir,
				Check:   opt.Check,
				Package: "server",
			})
			if err != nil {
				return nil, err
			}
			files = append(files, fs...)

		default:
			return nil, fmt.Errorf("unknown target: %s", t)
		}
	}
	return files, nil
}
