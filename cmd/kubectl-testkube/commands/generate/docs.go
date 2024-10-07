package generate

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func NewDocsCmd() *cobra.Command {
	filePrepender := func(filename string) string {
		return `
<head>
  <meta name="og:type" content="reference-doc" />
</head>

`
	}

	linkHandler := func(name string) string {
		return name
	}

	return &cobra.Command{
		Use:     "doc",
		Aliases: []string{"docs"},
		Short:   "Generate docs for kubectl testkube",
		Long:    `Generate docs for kubectl testkube`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			return doc.GenMarkdownTreeCustom(root, "gen/docs/cli", filePrepender, linkHandler)
		},
	}
}
