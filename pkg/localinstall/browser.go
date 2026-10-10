package localinstall

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// False means the user must open it themselves.
func OpenBrowser(url string) bool {
	kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	args := browserCommand(runtime.GOOS, os.Getenv, string(kernel), exec.LookPath, url)
	if args == nil {
		return false
	}
	// A stuck opener must not hang the end of the install.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := exec.CommandContext(ctx, args[0], args[1:]...).Run()
	// explorer.exe exits 1 even after opening the page.
	var exitErr *exec.ExitError
	return err == nil || (args[0] == "explorer.exe" && ctx.Err() == nil && errors.As(err, &exitErr) && exitErr.ExitCode() == 1)
}

func browserCommand(goos string, getenv func(string) string, kernel string, lookPath func(string) (string, error), url string) []string {
	switch {
	case goos == "darwin":
		return []string{"open", url}
	case goos == "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", url}
	// WSL has no Linux browser; open the Windows one.
	case isWSL(kernel):
		if _, err := lookPath("wslview"); err == nil {
			return []string{"wslview", url}
		}
		return []string{"explorer.exe", url}
	// Without a screen, xdg-open may start a text browser.
	case getenv("DISPLAY") == "" && getenv("WAYLAND_DISPLAY") == "":
		return nil
	default:
		return []string{"xdg-open", url}
	}
}
