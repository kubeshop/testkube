package commands

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/otiai10/copy"
)

func concat(args ...interface{}) []string {
	result := make([]string, 0)
	for _, a := range args {
		switch a := a.(type) {
		case string:
			result = append(result, a)
		case int:
			result = append(result, strconv.Itoa(a))
		case []string:
			result = append(result, a...)
		case []interface{}:
			result = append(result, concat(a...)...)
		}
	}
	return result
}

func Comm(cmd string, args ...interface{}) *exec.Cmd {
	return exec.Command(cmd, concat(args...)...)
}

func Run(c string, args ...interface{}) error {
	sub := Comm(c, args...)
	sub.Stdout = os.Stdout
	sub.Stderr = os.Stderr
	return sub.Run()
}

func RunWithRetry(retries int, delay time.Duration, c string, args ...interface{}) (err error) {
	for i := 0; i < retries; i++ {
		err = Run(c, args...)
		if err == nil {
			return nil
		}
		if i+1 < retries {
			nextDelay := time.Duration(i+1) * delay
			fmt.Printf("error, trying again in %s (attempt %d/%d): %s\n", nextDelay.String(), i+2, retries, err.Error())
			time.Sleep(nextDelay)
		}
	}
	return err
}

// copyDirContents copies directory contents from source to destination
func copyDirContents(src, dest string) error {
	fmt.Printf("📥 Moving the contents to %s...\n", dest)

	return copy.Copy(src, dest, copy.Options{
		OnError: func(srcPath, destPath string, err error) error {
			if err != nil {
				// Ignore chmod errors on mounted directories
				if srcPath == src && strings.Contains(err.Error(), "chmod") {
					return nil
				}
				fmt.Printf("warn: copying to %s: %s\n", destPath, err.Error())
			}
			return nil
		},
	})
}

// adjustFilePermissions ensures files have appropriate permissions
func adjustFilePermissions(path string) error {
	fmt.Printf("📥 Adjusting access permissions...\n")

	return filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		mode := info.Mode()
		// Ensure group has read/write permissions
		if mode.Perm()&0o060 != 0o060 {
			if err := os.Chmod(filePath, mode|0o060); err != nil {
				// Log but don't fail on permission errors
				fmt.Printf("warn: chmod %s: %s\n", filePath, err.Error())
			}
		}
		return nil
	})
}

// listDirectoryContents displays the contents of a directory
func listDirectoryContents(path string) error {
	fmt.Printf("🔎 Destination folder contains following files ...\n")

	return filepath.Walk(path, func(name string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Bold directory names
		if info.IsDir() {
			fmt.Printf("\x1b[1m%s\x1b[0m\n", name)
		} else {
			fmt.Println(name)
		}
		return nil
	})
}
