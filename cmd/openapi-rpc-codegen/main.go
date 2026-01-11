package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xxxbrian/openapi-rpc-codegen/pkg/codegen"
)

func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func main() {
	var (
		specPath = flag.String("spec", "", "Path to openapi.yaml or openapi.json (required)")
		outDir   = flag.String("out", ".", "Output directory")
		targets  = flag.String("targets", "ts-wx", "Comma-separated targets: ts-wx,go-server")
		baseURL  = flag.String("base-url", "", "Override servers[0].url")
		check    = flag.Bool("check", false, "Check-only mode: do not write, fail if output differs")
		verbose  = flag.Bool("v", false, "Verbose logs")
	)
	flag.Parse()

	if *specPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -spec is required")
		flag.Usage()
		os.Exit(2)
	}

	opts := codegen.Options{
		SpecPath: *specPath,
		OutDir:   *outDir,
		BaseURL:  *baseURL,
		Check:    *check,
		Verbose:  *verbose,
		Targets:  splitCSV(*targets),
	}

	res, err := codegen.Generate(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if opts.Verbose {
		fmt.Printf("Generated %d file(s)\n", len(res.Files))
		for _, f := range res.Files {
			fmt.Println(" -", f)
		}
	}
}
