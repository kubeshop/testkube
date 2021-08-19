package commands

import (
	"fmt"
	"strings"

	"github.com/kubeshop/kubtest/pkg/ui"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate docs for kubectl kubtest",
	Long:  `Generate docs for kubectl kubtest`,
	Run: func(cmd *cobra.Command, args []string) {

		root := cmd.Root()
		commands := root.Commands()

		code()
		ui.LogoNoColor()
		code()

		fmt.Printf("# Commands reference for `kubectl kubetest` plugin\n\n")

		fmt.Printf("Commands tree\n")
		code()
		printCommandsTree(0, commands)
		code()

		fmt.Print("\n\n")

		printCommands(0, commands)
	},
}

func code() {
	fmt.Println("```")
}

func printCommands(level int, commands []*cobra.Command) {
	for _, cmd := range commands {
		printCommand(level, cmd)

		if len(cmd.Commands()) > 0 {
			printCommands(level+1, cmd.Commands())
		}
	}
}

func printCommand(level int, cmd *cobra.Command) {

	header := strings.Repeat("#", level+2)
	fmt.Printf("%s `%s` command\n\n", header, cmd.Name())

	fmt.Printf("%+v\n", cmd.Example)

	code()
	cmd.Help()
	code()

	fmt.Printf("\n\n")

}

func printCommandsTree(level int, commands []*cobra.Command) {
	for _, cmd := range commands {
		printCommandNameWithIndent(level, cmd)

		if len(cmd.Commands()) > 0 {
			printCommandsTree(level+1, cmd.Commands())
		}
	}
}

func printCommandNameWithIndent(level int, cmd *cobra.Command) {
	indent := strings.Repeat("\t", level)
	fmt.Printf("%s%+v\n", indent, cmd.Name())
}
