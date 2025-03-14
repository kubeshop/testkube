package generate

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
---
`

var filePrepender = func(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
}

var linkHandler = func(name string) string {
	return name
}

func NewDocsCmd() *cobra.Command 
func NewDocsCmd() *cobra.Command {
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
