package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doc",
		Short: "Generate docs for kubectl kubtest",
		Long:  `Generate docs for kubectl kubtest`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			return doc.GenMarkdownTree(root, "docs/cli")
		},
	}
}
