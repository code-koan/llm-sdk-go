package main

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate missing _index.md from godoc",
	Long: `Scan directories with 3+ .go files and generate _index.md skeletons
	for any that are missing. Uses go/doc to extract package descriptions.

	Existing _index.md files are NEVER overwritten.`,
	RunE: runGen,
}

func runGen(cmd *cobra.Command, args []string) error {
	var created int

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if skipDir(name) {
			return filepath.SkipDir
		}

		files, _ := goFiles(path)
		if len(files) < 3 {
			return nil
		}

		idx, err := extract(path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  SKIP %s: %v\n", path, err)
			return nil
		}

		written, err := render(idx)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  FAIL %s: %v\n", path, err)
			return nil
		}
		if written {
			fmt.Printf("  NEW: %s/_index.md (%d files)\n", path, len(idx.Files))
			created++
		}
		return nil
	})
	if err != nil {
		return err
	}

	if created == 0 {
		fmt.Println("OK: all _index.md files already exist.")
	} else {
		fmt.Printf("\n%d new _index.md file(s) generated.\n", created)
	}
	return nil
}
