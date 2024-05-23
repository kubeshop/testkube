package config

import (
	"encoding/json"
	"os"
	"path"

	"github.com/kubeshop/testkube/pkg/oauth"
)

const (
	APIServerName               string = "testkube-api-server"
	APIServerPort               int    = 8088
	DashboardName               string = "testkube-dashboard"
	DashboardPort               int    = 8080
	EnterpriseUiName            string = "testkube-enterprise-ui"
	EnterpriseUiPort            int    = 8080
	EnterpriseApiName           string = "testkube-enterprise-api"
	EnterpriseApiPort           int    = 8088
	EnterpriseApiForwardingPort int    = 8090
	EnterpriseDexName           string = "testkube-enterprise-dex"
	EnterpriseDexPort           int    = 5556
	EnterpriseDexForwardingPort int    = 5556

	configDirName = ".testkube"
	configFile    = "config.json"
)

var DefaultConfig = Data{
	TelemetryEnabled: true,
	Namespace:        "testkube",
	APIURI:           "http://localhost:8088",
	APIServerName:    APIServerName,
	APIServerPort:    APIServerPort,
	DashboardName:    DashboardName,
	DashboardPort:    DashboardPort,
	OAuth2Data: OAuth2Data{
		Provider: oauth.GithubProviderType,
	},
}

func GetStorage(dir string) (Storage, error) {
	storage := Storage{Dir: dir}
	err := storage.Init()
	return storage, err
}

type Storage struct {
	Dir string
}

func (c *Storage) Load() (data Data, err error) {
	path, err := c.getPath()
	if err != nil {
		return data, err
	}
	d, err := os.ReadFile(path)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(d, &data)
	return data, err
}

func (c *Storage) Save(data Data) error {
	d, err := json.Marshal(data)
	if err != nil {
		return err
	}
	path, err := c.getPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, d, 0700)
}

func (c *Storage) Init() error {
	var defaultConfig = DefaultConfig
	// create ConfigWriter dir if not exists
	dir, err := c.getDir()
	if err != nil {
		return err
	}
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// create empty JSON file if not exists
	path, err := c.getPath()
	if err != nil {
		return err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		f.Close()

		return c.Save(defaultConfig)
	} else if err != nil {
		return err
	}

	return nil
}

func (c *Storage) getDir() (string, error) {
	var err error
	var dir string
	if c.Dir != "" {
		dir = c.Dir
	} else {
		dir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}

	return path.Join(dir, configDirName), nil
}

func (c *Storage) getPath() (string, error) {
	dir, err := c.getDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, configFile), nil
}
