package commands

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/pkg/localinstall"
	"github.com/kubeshop/testkube/pkg/ui"
)

const statusLocalCommand = "testkube status local"

// Distinct codes let scripts tell the states apart; 1 means error.
const (
	statusExitStarting     = 2
	statusExitNotRunning   = 3
	statusExitDockerAbsent = 4
)

func NewStatusLocalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Show whether the local Testkube trial runs, and its login",
		// Hidden until the guided installer is complete.
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runStatusLocal(cmd.Context()))
		},
	}
	hideGlobalFlags(cmd)
	return cmd
}

func runStatusLocal(ctx context.Context) int {
	// Install only runs in WSL, so its cluster lives there.
	if runtime.GOOS == "windows" {
		ui.Printf("  On Windows the local trial lives inside WSL. In your WSL terminal, run:\n    %s\n", statusLocalCommand)
		return 1
	}
	s := localinstall.ReadStatus(ctx)
	ok, notOk := ui.Green("✔"), ui.LightRed("✖")
	row := func(icon, name, detail string) { ui.Printf("  %s %-10s %s\n", icon, name, detail) }
	next := func(lead, command string) { ui.Printf("\n  %s\n    %s\n", lead, command) }
	switch s.State {
	case localinstall.TrialNoDocker:
		row(notOk, "docker", s.Docker.Detail)
		next("Start Docker, then run again:", statusLocalCommand)
		return statusExitDockerAbsent
	case localinstall.TrialNotInstalled:
		if !s.DataKept {
			row(ui.LightGray("○"), "cluster", "not installed")
			next("To try Testkube on this machine, run:", localinstall.InstallCommand)
			return statusExitNotRunning
		}
		row(ok, "docker", "running")
		row(ui.LightGray("○"), "cluster", "not installed")
		row(ok, "data", "kept in ~/.testkube/data")
		next("To bring it back, run:", localinstall.InstallCommand)
		return statusExitNotRunning
	case localinstall.TrialStopped:
		row(ok, "docker", "running")
		row(notOk, "cluster", "stopped")
		next("To start it again, run:", localinstall.InstallCommand)
		return statusExitNotRunning
	case localinstall.TrialStarting:
		row(ok, "docker", "running")
		row(ok, "cluster", "running")
		row(ui.LightCyan("⠼"), "testkube", "still starting")
		next("Check again in a minute:", statusLocalCommand)
		return statusExitStarting
	}
	row(ok, "docker", "running")
	row(ok, "cluster", "running")
	row(ok, "testkube", "ready")
	label := func(s string) string { return ui.LightGray(fmt.Sprintf("%-12s", s)) }
	ui.Printf("\n  %s%s\n  %s%s\n  %s%s\n",
		label("Dashboard"), fmt.Sprintf("http://localhost:%d", s.Ports["dashboard"]),
		label("Email"), localinstall.RememberLoginEmail(""),
		label("Password"), localinstall.AdminPassword)
	return 0
}
