// index.go — shared types, go/doc extractor, and template renderer
// for _index.md generation.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// FileEntry describes a single Go source file in a package directory.
type FileEntry struct {
	Name string // filename, e.g. "anthropic.go"
	Role string // human label, defaults to filename minus .go
}

// Index holds everything needed to render an _index.md for one directory.
type Index struct {
	Dir         string      // relative path, e.g. "providers/anthropic"
	Name        string      // directory basename
	Description string      // from go/doc.Synopsis, or fallback
	Files       []FileEntry // non-test, non-gen .go files
}

// Gap describes a missing _index.md coverage item.
type Gap struct {
	Dir    string
	Reason string // "missing _index.md", "not in docs/_index.md", "not in providers/_index.md"
}

// skipDir reports whether a directory should not be scanned.
func skipDir(name string) bool {
	if name == "." {
		return false
	}
	switch name {
	case ".git", ".claude", ".codegraph", ".specify",
		"cmd", "vendor", "testdata", "graphify-out", "docs",
		"examples", "node_modules":
		return true
	}
	return strings.HasPrefix(name, ".") && len(name) > 1
}

// goFiles returns non-test, non-gen .go file entries in dir.
func goFiles(dir string) ([]FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []FileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasPrefix(n, ".") {
			continue
		}
		if !strings.HasSuffix(n, ".go") {
			continue
		}
		if strings.HasSuffix(n, "_test.go") || strings.HasSuffix(n, "_gen.go") {
			continue
		}
		files = append(files, FileEntry{
			Name: n,
			Role: strings.TrimSuffix(n, ".go"),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

// extract reads Go source from absDir and builds an Index using go/doc.
func extract(absDir string) (*Index, error) {
	dirName := filepath.Base(absDir)
	idx := &Index{
		Dir:  absDir,
		Name: dirName,
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, absDir, func(fi os.FileInfo) bool {
		n := fi.Name()
		return !strings.HasSuffix(n, "_test.go") &&
			!strings.HasSuffix(n, "_gen.go") &&
			strings.HasSuffix(n, ".go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", absDir, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages found in %s", absDir)
	}

	// Pick package matching directory name, or first available.
	for pname, p := range pkgs {
		if pname == dirName || pname == dirName+"_test" {
			idx.Name = pname
			return buildIndex(idx, p), nil
		}
	}
	// Fallback: use first package found (name may differ from dir)
	for pname, p := range pkgs {
		idx.Name = pname
		return buildIndex(idx, p), nil
	}
	return nil, fmt.Errorf("no package found in %s", absDir)
}

func buildIndex(idx *Index, pkg *ast.Package) *Index {
	d := doc.New(pkg, idx.Name, doc.AllDecls)
	desc := doc.Synopsis(d.Doc)
	if desc == "" {
		desc = idx.Name + " package"
	}
	idx.Description = desc

	files, err := goFiles(idx.Dir)
	if err == nil {
		idx.Files = files
	}
	return idx
}

const indexTmpl = `---
description: {{.Description}}
---

# {{.Name}}

{{.Description}}.

## 核心文件

| 文件 | 职责 |
|------|------|
{{range .Files}}| ` + "`{{.Name}}`" + ` | {{.Role}} |
{{end}}`

var tmpl = template.Must(template.New("index").Parse(indexTmpl))

// render writes idx as _index.md. Returns false if file already exists.
func render(idx *Index) (bool, error) {
	target := filepath.Join(idx.Dir, "_index.md")
	if _, err := os.Stat(target); err == nil {
		return false, nil // already exists, never overwrite
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, idx); err != nil {
		return false, err
	}
	if err := os.WriteFile(target, buf.Bytes(), 0o644); err != nil {
		return false, err
	}
	return true, nil
}
