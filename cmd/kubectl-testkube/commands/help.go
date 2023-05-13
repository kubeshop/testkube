package commands

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/ui"
)

const (
	cmdGroupAnnotation = "GroupAnnotation"
	cmdMngmCmdGroup    = "1-Management commands"
	cmdGroupCommands   = "2-Commands"
	cmdGroupCobra      = "other"

	cmdGroupDelimiter = "-"
)

func NewHelpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "Help about any command",
		Long:  "Display the available commands and flags",
		Run: func(cmd *cobra.Command, args []string) {
			ui.Print(RootCmd.Short)
			ui.NL()
			ui.Print(ui.LightGray("Usage"))
			ui.Printf(fmt.Sprintf("%s %s", ui.White(RootCmd.Use), ui.LightGray("[flags]")))
			ui.Printf("%s %s", ui.White(RootCmd.Use), ui.LightGray("[command]"))
			ui.NL()
			usage := helpMessageByGroups(RootCmd)
			ui.Print(usage)
			ui.Print(ui.LightGray("Flags"))
			ui.Printf(RootCmd.Flags().FlagUsages())
			ui.Print(ui.LightGray("Use \"kubectl testkube [command] --help\" for more information about a command."))
			ui.NL()
			ui.Printf("%s   %s", ui.LightGray("Docs & Support:"), ui.White("https://docs.testkube.io"))
			ui.NL()
		},
	}
}

func helpMessageByGroups(cmd *cobra.Command) string {

	groups := map[string][]string{}
	for _, c := range cmd.Commands() {
		var groupName string
		v, ok := c.Annotations[cmdGroupAnnotation]
		if !ok {
			groupName = cmdGroupCobra
		} else {
			groupName = v
		}

		groupCmds := groups[groupName]
		groupCmds = append(groupCmds, fmt.Sprintf("%-16s%s", c.Name(), ui.LightGray(c.Short)))
		sort.Strings(groupCmds)

		groups[groupName] = groupCmds
	}

	if len(groups[cmdGroupCobra]) != 0 {
		groups[cmdMngmCmdGroup] = append(groups[cmdMngmCmdGroup], groups[cmdGroupCobra]...)
	}
	delete(groups, cmdGroupCobra)

	groupNames := []string{}
	for k := range groups {
		groupNames = append(groupNames, k)
	}
	sort.Strings(groupNames)

	buf := bytes.Buffer{}
	for _, groupName := range groupNames {
		commands := groups[groupName]

		groupSplit := strings.Split(groupName, cmdGroupDelimiter)
		group := "others"
		if len(groupSplit) > 1 {
			group = strings.Split(groupName, cmdGroupDelimiter)[1]
		}
		buf.WriteString(fmt.Sprintf("%s\n", ui.LightGray(group)))

		for _, cmd := range commands {
			buf.WriteString(fmt.Sprintf("%s\n", cmd))
		}
		buf.WriteString("\n")
	}
	return buf.String()
}
