package generate

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doc",
		Aliases: []string{"docs"},
		Short:   "Generate docs for kubectl testkube",
		Long:    `Generate docs for kubectl testkube`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			return doc.GenMarkdownTree(root, "docs/docs/cli")
		},
	}
}
