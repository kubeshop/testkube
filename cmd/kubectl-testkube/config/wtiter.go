package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

const dir = ".testkube"
const fileName = "config.json"

type Storage struct {
}

func (c *Storage) Load() (data Data, err error) {
	d, err := ioutil.ReadFile(path.Join(c.getDir(), fileName))
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
	return ioutil.WriteFile(path.Join(c.getDir(), fileName), d, 0700)
}

func (c *Storage) Init() error {
	// create ConfigWriter dir if not exists
	if dir := c.getDir(); dir != "" {
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			return os.Mkdir(dir, 0700)
		}
	}
	return nil
}

func (c *Storage) getDir() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return path.Join(dirname, dir)

}
