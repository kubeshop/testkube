package localinstall

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// False means the user must open it themselves.
func OpenBrowser(url string) bool {
	kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	args := browserCommand(runtime.GOOS, os.Getenv, string(kernel), exec.LookPath, url)
	if args == nil {
		return false
	}
	err := exec.Command(args[0], args[1:]...).Run()
	// explorer.exe exits 1 even after opening the page.
	var exitErr *exec.ExitError
	return err == nil || (args[0] == "explorer.exe" && errors.As(err, &exitErr))
}

func browserCommand(goos string, getenv func(string) string, kernel string, lookPath func(string) (string, error), url string) []string {
	switch {
	case goos == "darwin":
		return []string{"open", url}
	case goos == "windows":
		return []string{"rundll32", "url.dll,FileProtocolHandler", url}
	// WSL has no Linux browser; open the Windows one.
	case strings.Contains(strings.ToLower(kernel), "microsoft"):
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
