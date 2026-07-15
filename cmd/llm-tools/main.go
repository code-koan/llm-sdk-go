package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"

	"github.com/code-koan/llm-sdk-go/internal/codegen"
)

var stdout io.Writer = os.Stdout

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("llm-tools", flag.ContinueOnError)
	source := fs.String("source", "", "Input Go source file (required)")
	output := fs.String("output", "", "Output directory (default: same directory as source)")
	dryRun := fs.Bool("dry-run", false, "Print generated code to stdout instead of writing file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *source == "" {
		return fmt.Errorf("-source is required")
	}

	sourcePath, err := filepath.Abs(*source)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}

	// Extract package name from source file.
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, sourcePath, nil, parser.PackageClauseOnly)
	if err != nil {
		return fmt.Errorf("parse source: %w", err)
	}
	pkgName := f.Name.Name

	svc, err := codegen.ParseService(sourcePath)
	if err != nil {
		return fmt.Errorf("parse service: %w", err)
	}

	if *dryRun {
		return codegen.Generate(stdout, pkgName, svc)
	}

	outputDir := *output
	if outputDir == "" {
		outputDir = filepath.Dir(sourcePath)
	}

	outputPath := filepath.Join(outputDir, codegen.SnakeCase(svc.Name)+".gen.go")
	return codegen.GenerateFile(outputPath, pkgName, svc)
}
