package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "tools",
		Short: "llm-sdk-go development tools",
	}

	index := &cobra.Command{
		Use:   "index",
		Short: "_index.md coverage check & generation",
	}
	index.AddCommand(checkCmd, genCmd)

	root.AddCommand(index)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
