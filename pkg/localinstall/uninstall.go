package localinstall

import (
	"os"
	"path/filepath"
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
