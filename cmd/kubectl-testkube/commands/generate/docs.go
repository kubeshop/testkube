package generate

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
title: "%s"
%s---
<head>
  <meta name="docsearch:indexPrefix" content="reference-doc" />
</head>

`

var filePrepender = func(filename string) string {
	name := filepath.Base(filename)
	base := strings.TrimSuffix(name, path.Ext(name))
	sidebarPosition := ""
	if strings.EqualFold(base, "testkube") {
		sidebarPosition = "sidebar_position: 1\n"
	}

	return fmt.Sprintf(fmTemplate, strings.Replace(base, "-", " ", -1), sidebarPosition)
}

var linkHandler = func(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "_", "-"), ".md", ".mdx")
}

func NewDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "doc",
		Aliases: []string{"docs"},
		Short:   "Generate docs for kubectl testkube",
		Long:    `Generate docs for kubectl testkube`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			root.DisableAutoGenTag = true
			return GenTestkubeMarkdownTree(root, "gen/docs/cli", filePrepender, linkHandler)
		},
	}
}

// GenMarkdownTreeCustom is the same as GenMarkdownTree, but
// with custom filePrepender and linkHandler.
func GenTestkubeMarkdownTree(cmd *cobra.Command, dir string, filePrepender, linkHandler func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := GenTestkubeMarkdownTree(c, dir, filePrepender, linkHandler); err != nil {
			return err
		}
	}

	basename := strings.ReplaceAll(cmd.CommandPath(), " ", "-") + ".mdx"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	if err := doc.GenMarkdownCustom(cmd, f, linkHandler); err != nil {
		return err
	}
	return nil
}
