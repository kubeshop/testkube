package localinstall

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Only the cluster's files; data, tools and reports stay.
func RemoveFiles() (removed bool, err error) {
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return false, err
	}
	return removeFiles(filepath.Dir(kubeconfig))
}

func removeFiles(dir string) (removed bool, err error) {
	paths := []string{filepath.Join(dir, "kind.yaml"), filepath.Join(dir, "kubeconfig"), filepath.Join(dir, "helm")}
	// A killed install can leave the license key in these.
	temp, _ := filepath.Glob(filepath.Join(dir, "install-*"))
	paths = append(paths, temp...)
	// Last: a rerun after a failure must still see ours.
	paths = append(paths, filepath.Join(dir, "cluster-id"))
	for _, p := range paths {
		if _, err := os.Lstat(p); err != nil {
			continue
		}
		if err := os.RemoveAll(p); err != nil {
			return removed, err
		}
		removed = true
	}
	return removed, nil
}

// HasData reports kept data a reinstall would reuse.
func HasData() bool {
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return false
	}
	entries, _ := os.ReadDir(filepath.Join(filepath.Dir(kubeconfig), "data", Namespace))
	return len(entries) > 0
}

var ErrDataUnsafe = errors.New("the data folder isn't a plain folder we made")

// Run after the cluster is gone: nothing may write meanwhile.
func DeleteData(ctx context.Context) (removed bool, sudo string, err error) {
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return false, "", err
	}
	return deleteData(ctx, filepath.Dir(kubeconfig), runCombined)
}

func deleteData(ctx context.Context, dir string, run runFunc) (removed bool, sudo string, err error) {
	data := filepath.Join(dir, "data")
	secrets := filepath.Join(dir, "secrets.json")
	info, err := os.Lstat(data)
	switch {
	// A leftover passwords file goes quietly; there was no data.
	case errors.Is(err, fs.ErrNotExist):
		removeIfThere(secrets)
		return false, "", nil
	case err != nil:
		return false, "", err
	// Docker follows a symlinked mount source: never delete through one.
	case !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 || filepath.Base(dir) != ".testkube":
		return false, "", ErrDataUnsafe
	}
	entries, err := os.ReadDir(data)
	if err != nil {
		return false, "", err
	}
	// On Linux containers own some files; rm alone may fail.
	if len(entries) > 0 {
		args := []string{"run", "--rm", "--pull=never", "--network=none", "--entrypoint", "rm",
			"-v", data + ":/d", nodeImage, "-rf", "--one-file-system"}
		for _, e := range entries {
			args = append(args, "/d/"+e.Name())
		}
		_, _ = run(ctx, "docker", args...)
	}
	if err := os.RemoveAll(data); err != nil {
		return false, "sudo rm -rf " + shellQuote(data), err
	}
	// Only now: passwords without data are harmless, the reverse isn't.
	removeIfThere(secrets)
	return true, "", nil
}

// Single quotes stop expansion; an inner quote needs closing first.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func removeIfThere(path string) bool {
	if _, err := os.Lstat(path); err != nil {
		return false
	}
	return os.Remove(path) == nil
}
