package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v2"
)

type HelmChart yaml.MapSlice

func Read(filePath string) (helmChart HelmChart, err error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return helmChart, err
	}

	err = yaml.Unmarshal(content, &helmChart)
	return
}

func Write(filePath string, helmChart HelmChart) (err error) {

	content, err := yaml.Marshal(helmChart)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, content, 0644)
}

func GetVersion(helmChart HelmChart) string {
	// go through SliceMap
	for _, i := range helmChart {
		if i.Key == "version" {
			val, ok := i.Value.(string)
			if ok {
				return val
			}
		}
	}

	return "0.0.0"
}

func SaveString(helmChart *HelmChart, key, value string) error {
	for k := range *helmChart {
		if (*helmChart)[k].Key == key {
			(*helmChart)[k].Value = value
			return nil
		}
	}

	return fmt.Errorf("key %s not found in %+v", key, helmChart)
}

func GetChart(dir string) (helmChart HelmChart, chartPath string, err error) {
	chartPath, err = Find(dir)
	if err != nil {
		return helmChart, chartPath, err
	}

	helmChart, err = Read(chartPath)
	return helmChart, chartPath, err
}

func Find(dir string) (chartPath string, err error) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !dirInfo.IsDir() {
		return "", fmt.Errorf("passed '%s' path is not directory", dir)
	}

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if info.Name() == "Chart.yaml" {
			chartPath = path
			return filepath.SkipDir
		}

		return nil
	})

	return
}

func UpdateValuesImageTag(path, tag string) error {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	r := regexp.MustCompile(`tag: "[^"]+"`)
	output := r.ReplaceAll(input, []byte(fmt.Sprintf(`tag: "%s"`, tag)))

	return ioutil.WriteFile(path, output, 0644)
}
