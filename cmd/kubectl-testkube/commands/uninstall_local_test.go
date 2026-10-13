package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalHelp_HidesFlagsThatDoNothingHere(t *testing.T) {
	for _, args := range [][]string{{"install", "local", "--help"}, {"uninstall", "local", "--help"}} {
		root := &cobra.Command{Use: "testkube"}
		root.PersistentFlags().String("namespace", "tk-dev", "")
		install := &cobra.Command{Use: "install"}
		install.AddCommand(NewInstallLocalCmd())
		root.AddCommand(install, NewPurgeCmd())
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs(args)

		require.NoError(t, root.Execute())
		assert.NotContains(t, out.String(), "--namespace", args[0])
		assert.Contains(t, out.String(), "Examples:", args[0])
	}
}

func TestUninstallLocal_NeverReachesTheRealClusterPurge(t *testing.T) {
	root := &cobra.Command{Use: "testkube"}
	root.AddCommand(NewPurgeCmd())
	tests := map[string]struct {
		args []string
		want string
	}{
		"uninstall local is ours":      {[]string{"uninstall", "local"}, "local"},
		"purge local is ours":          {[]string{"purge", "local"}, "local"},
		"plain uninstall still purges": {[]string{"uninstall"}, "purge"},
		"other words still purge":      {[]string{"purge", "something"}, "purge"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cmd, _, err := root.Find(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.want, cmd.Name())
		})
	}
}

func TestShouldDeleteData_NoAnswerMeansNo(t *testing.T) {
	tests := map[string]struct {
		yes, interactive, answer bool
		want                     bool
		wantErr                  error
		asked                    bool
	}{
		"--yes skips the question":  {yes: true, want: true},
		"no terminal needs --yes":   {wantErr: errNeedsYes},
		"declined keeps everything": {interactive: true, asked: true},
		"accepted deletes":          {interactive: true, answer: true, want: true, asked: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			asked := false

			got, err := shouldDeleteData(tt.yes, tt.interactive, func() bool { asked = true; return tt.answer })

			assert.Equal(t, tt.want, got)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.asked, asked)
		})
	}
}
