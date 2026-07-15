package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check _index.md coverage",
	Long: `Scan codebase for _index.md coverage gaps:

	  - Directories with 3+ .go files missing _index.md
	  - docs/*.md not referenced in docs/_index.md
	  - providers/*/ not listed in providers/_index.md`,
	RunE: runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	var gaps []Gap

	// 1. Scan for directories with 3+ .go files lacking _index.md
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if skipDir(name) {
			return filepath.SkipDir
		}
		// Count non-test, non-gen .go files
		files, err := goFiles(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARN: %s: %v\n", path, err)
			return nil
		}
		if len(files) >= 3 {
			if _, err := os.Stat(filepath.Join(path, "_index.md")); os.IsNotExist(err) {
				gaps = append(gaps, Gap{Dir: path, Reason: "missing _index.md"})
			}
		}
		return nil
	})

	// 2. Check docs/_index.md references
	if data, err := os.ReadFile("docs/_index.md"); err == nil {
		content := string(data)
		filepath.WalkDir("docs", func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".md") || strings.HasSuffix(path, "_index.md") {
				return nil
			}
			if strings.Contains(path, "/reference/") {
				return nil
			}
			basename := strings.TrimSuffix(filepath.Base(path), ".md")
			if !strings.Contains(content, basename) {
				gaps = append(gaps, Gap{Dir: path, Reason: "not in docs/_index.md"})
			}
			return nil
		})
	} else if os.IsNotExist(err) {
		gaps = append(gaps, Gap{Dir: "docs", Reason: "_index.md is missing"})
	}

	// 3. Check providers/_index.md listing
	if entries, err := os.ReadDir("providers"); err == nil {
		if data, err := os.ReadFile("providers/_index.md"); err == nil {
			content := string(data)
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				name := e.Name()
				if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
					continue
				}
				// Must have .go files to be a provider
				hasGo := false
				filepath.Walk(filepath.Join("providers", name), func(path string, info os.FileInfo, err error) error {
					if hasGo {
						return filepath.SkipAll
					}
					if strings.HasSuffix(path, ".go") {
						hasGo = true
					}
					return nil
				})
				if hasGo && !strings.Contains(content, name+"/") {
					gaps = append(gaps, Gap{Dir: "providers/" + name, Reason: "not in providers/_index.md"})
				}
			}
		} else if os.IsNotExist(err) {
			gaps = append(gaps, Gap{Dir: "providers", Reason: "_index.md is missing"})
		}
	}

	if len(gaps) > 0 {
		for _, g := range gaps {
			fmt.Printf("  %s: %s\n", g.Dir, g.Reason)
		}
		return fmt.Errorf("\nFAIL: %d _index.md gap(s) found", len(gaps))
	}

	fmt.Println("OK: _index.md coverage complete.")
	return nil
}
