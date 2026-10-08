package localinstall

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kubeshop/testkube/pkg/utils"
)

const Namespace = "testkube"

var (
	ErrSecretsLost    = errors.New("data from an earlier install exists but its saved passwords are gone")
	ErrSecretsDamaged = errors.New("the saved passwords can't be read")
)

// On the host, so rebuilt clusters open the kept data.
type Secrets struct {
	RunnerKey      string `json:"runnerKey"`
	MasterPassword string `json:"masterPassword"`
	MinioPassword  string `json:"minioPassword"`
	AIToken        string `json:"aiToken"`
}

func LoadOrCreateSecrets() (Secrets, error) {
	kubeconfig, err := KubeconfigPath()
	if err != nil {
		return Secrets{}, err
	}
	return loadOrCreateSecrets(filepath.Dir(kubeconfig))
}

func loadOrCreateSecrets(dir string) (Secrets, error) {
	path := filepath.Join(dir, "secrets.json")
	data, err := os.ReadFile(path)
	if err == nil {
		var s Secrets
		if err := json.Unmarshal(data, &s); err != nil || s.RunnerKey == "" || s.MasterPassword == "" ||
			s.MinioPassword == "" || s.AIToken == "" {
			return Secrets{}, fmt.Errorf("%w: %s", ErrSecretsDamaged, path)
		}
		return s, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return Secrets{}, err
	}
	// New passwords can't open data sealed with the old ones.
	if entries, _ := os.ReadDir(filepath.Join(dir, "data", Namespace)); len(entries) > 0 {
		return Secrets{}, ErrSecretsLost
	}
	s := Secrets{
		RunnerKey:      "tkckey_agent_" + utils.RandAlphanum(32),
		MasterPassword: utils.RandAlphanum(32),
		MinioPassword:  utils.RandAlphanum(32),
		AIToken:        utils.RandAlphanum(32),
	}
	data, err = json.MarshalIndent(s, "", "  ")
	if err != nil {
		return Secrets{}, err
	}
	return s, writePrivate(path, data)
}

// CreateTemp is 0600; rename never leaves a half-written file.
func writePrivate(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
