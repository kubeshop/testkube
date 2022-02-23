package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

const configDirName = ".testkube"
const configFile = "config.json"

type Storage struct {
}

func (c *Storage) Load() (data Data, err error) {
	d, err := ioutil.ReadFile(c.getPath())
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
	return ioutil.WriteFile(c.getPath(), d, 0700)
}

func (c *Storage) Init() error {
	var defaultConfig = Data{AnalyticsEnabled: true}
	// create ConfigWriter dir if not exists
	dir := c.getDir()
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.Mkdir(dir, 0700)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// create empty JSON file if not exists
	path := c.getPath()
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		return c.Save(defaultConfig)
	} else if err != nil {
		return err
	}

	return nil
}

func (c *Storage) getDir() string {
	home, _ := os.UserHomeDir()
	return path.Join(home, configDirName)
}

func (c *Storage) getPath() string {
	return path.Join(c.getDir(), configFile)
}
