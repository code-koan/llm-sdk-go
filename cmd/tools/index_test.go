package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractDescription(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "doc.go"),
		[]byte("// Package testpkg provides awesome testing utilities.\n//\n// More detail.\npackage testpkg"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "testpkg.go"),
		[]byte("package testpkg\n\nfunc Foo() {}"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "testpkg_test.go"),
		[]byte("package testpkg\n\nimport \"testing\"\nfunc TestFoo(t *testing.T) {}"),
		0o644,
	))

	idx, err := extract(dir)
	require.NoError(t, err)
	require.Equal(t, "testpkg", idx.Name)
	require.Equal(t, "Package testpkg provides awesome testing utilities.", idx.Description)
	// test_test.go must be excluded
	require.Len(t, idx.Files, 2) // doc.go + testpkg.go
}

func TestExtractNoDocGo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "mylib.go"),
		[]byte("// Package mylib does stuff.\npackage mylib\n\nfunc Bar() {}"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "mylib_test.go"),
		[]byte("package mylib\n\nimport \"testing\"\nfunc TestBar(t *testing.T) {}"),
		0o644,
	))

	idx, err := extract(dir)
	require.NoError(t, err)
	require.Equal(t, "mylib", idx.Name)
	// Should extract from mylib.go's package comment
	require.Contains(t, idx.Description, "does stuff")
	require.Len(t, idx.Files, 1) // mylib.go only
}

func TestExtractEmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	_, err := extract(dir)
	require.Error(t, err)
}

func TestRenderDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "doc.go"),
		[]byte("// Package keepme provides original content.\npackage keepme"),
		0o644,
	))

	idx, err := extract(dir)
	require.NoError(t, err)

	// First render succeeds
	written, err := render(idx)
	require.NoError(t, err)
	require.True(t, written)

	// Second render should NOT overwrite
	written, err = render(idx)
	require.NoError(t, err)
	require.False(t, written)

	// Content preserved
	data, err := os.ReadFile(filepath.Join(dir, "_index.md"))
	require.NoError(t, err)
	require.Contains(t, string(data), "provides original content")
}

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "doc.go"),
		[]byte("// Package demo provides demo features.\npackage demo"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "demo.go"),
		[]byte("package demo"),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "helper.go"),
		[]byte("package demo"),
		0o644,
	))

	idx := &Index{
		Dir:         dir,
		Name:        "demo",
		Description: "provides demo features",
		Files: []FileEntry{
			{Name: "demo.go", Role: "demo"},
			{Name: "helper.go", Role: "helper"},
		},
	}

	written, err := render(idx)
	require.NoError(t, err)
	require.True(t, written)

	data, err := os.ReadFile(filepath.Join(dir, "_index.md"))
	require.NoError(t, err)
	content := string(data)

	require.Contains(t, content, "description: provides demo features")
	require.Contains(t, content, "`demo.go`")
	require.Contains(t, content, "`helper.go`")
}

func TestGoFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{
		"real.go", "real2.go", "test_test.go", "gen_gen.go", "README.md", ".hidden.go",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("package x"), 0o644))
	}

	files, err := goFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2) // real.go + real2.go only
}
