package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// MapItem is a key-value pair in an ordered YAML mapping.
type MapItem struct {
	Key, Value interface{}
}

// HelmChart is an ordered list of key-value pairs that represents a YAML mapping,
// preserving the original key order when marshaling/unmarshaling.
type HelmChart []MapItem

// MarshalYAML implements yaml.Marshaler for HelmChart so that it is serialized
// as a YAML mapping node (block style) with keys in their original order.
func (hc HelmChart) MarshalYAML() (interface{}, error) {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, item := range hc {
		keyNode := new(yaml.Node)
		if err := keyNode.Encode(item.Key); err != nil {
			return nil, err
		}
		if keyNode.Kind == yaml.DocumentNode && len(keyNode.Content) > 0 {
			keyNode = keyNode.Content[0]
		}
		valNode := new(yaml.Node)
		if err := valNode.Encode(item.Value); err != nil {
			return nil, err
		}
		if valNode.Kind == yaml.DocumentNode && len(valNode.Content) > 0 {
			valNode = valNode.Content[0]
		}
		node.Content = append(node.Content, keyNode, valNode)
	}
	return node, nil
}

// UnmarshalYAML implements yaml.Unmarshaler for HelmChart so that nested YAML
// mappings are also decoded as HelmChart (preserving key order).
func (hc *HelmChart) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("helm chart: expected a YAML mapping, got node kind %v", value.Kind)
	}
	*hc = make(HelmChart, 0, len(value.Content)/2)
	for i := 0; i+1 < len(value.Content); i += 2 {
		var key interface{}
		if err := value.Content[i].Decode(&key); err != nil {
			return err
		}
		val, err := decodeHelmChartValue(value.Content[i+1])
		if err != nil {
			return err
		}
		*hc = append(*hc, MapItem{Key: key, Value: val})
	}
	return nil
}

// decodeHelmChartValue recursively decodes a yaml.Node so that nested mappings
// become HelmChart values (preserving key order) and sequences become []interface{}.
func decodeHelmChartValue(node *yaml.Node) (interface{}, error) {
	switch node.Kind {
	case yaml.MappingNode:
		hc := HelmChart{}
		if err := hc.UnmarshalYAML(node); err != nil {
			return nil, err
		}
		return hc, nil
	case yaml.SequenceNode:
		result := make([]interface{}, 0, len(node.Content))
		for _, child := range node.Content {
			v, err := decodeHelmChartValue(child)
			if err != nil {
				return nil, err
			}
			result = append(result, v)
		}
		return result, nil
	default:
		var val interface{}
		if err := node.Decode(&val); err != nil {
			return nil, err
		}
		return val, nil
	}
}

// Read reads helm chart based on path
func Read(filePath string) (helmChart HelmChart, err error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return helmChart, err
	}

	err = yaml.Unmarshal(content, &helmChart)
	return
}

// Write writes content of HelmChart to given path
func Write(filePath string, helmChart HelmChart) (err error) {

	content, err := yaml.Marshal(helmChart)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, content, 0644)
}

// UpdateDependencyVersion updates version in HelmChart
func UpdateDependencyVersion(in HelmChart, dependency, version string) (out HelmChart, err error) {
	// go through SliceMap
	for ci, i := range in {
		if i.Key == "dependencies" {
			deps, ok := i.Value.([]interface{})
			if !ok {
				return out, fmt.Errorf("dependencies key is not array")
			}

			for di, idep := range deps {
				fields, ok := idep.(HelmChart)
				if !ok {
					return out, fmt.Errorf("invalid dependencies key values")
				}

				for _, f := range fields {
					if f.Key == "name" && f.Value == dependency {
						for fi, f := range fields {
							if f.Key == "version" {
								in[ci].Value.([]interface{})[di].(HelmChart)[fi].Value = version
								return in, nil
							}

						}
					}
				}
			}
		}
	}
	return out, fmt.Errorf("dependency not found")
}

// GetDependencyVersion returns selected helm chart dependency version
func GetDependencyVersion(helmChart HelmChart, dependency string) (string, error) {
	for _, i := range helmChart {
		if i.Key == "dependencies" {
			deps, ok := i.Value.([]interface{})
			if !ok {
				return "", fmt.Errorf("dependencies key is not array")
			}

			for _, ifields := range deps {
				fields, ok := ifields.(HelmChart)
				if !ok {
					return "", fmt.Errorf("invalid dependencies key values")
				}

				for _, f := range fields {
					if f.Key == "name" && f.Value == dependency {
						for _, f := range fields {
							if f.Key == "version" {
								return f.Value.(string), nil
							}

						}
					}
				}
			}
		}
	}
	return "", fmt.Errorf("version key not found in dependency %s", dependency)
}

// GetVersion returns HelmChart version
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

// SaveString saves given key with passed value in HelmChart
func SaveString(helmChart *HelmChart, key, value string) error {
	for k := range *helmChart {
		if (*helmChart)[k].Key == key {
			(*helmChart)[k].Value = value
			return nil
		}
	}

	return fmt.Errorf("key %s not found in %+v", key, helmChart)
}

// GetChart returns Chart based on directory, locates chart automatically
func GetChart(dir string) (helmChart HelmChart, chartPath string, err error) {
	chartPath, err = Find(dir)
	if err != nil {
		return helmChart, chartPath, err
	}

	helmChart, err = Read(chartPath)
	return helmChart, chartPath, err
}

// Find locates helm chart in given dir
func Find(dir string) (chartPath string, err error) {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !dirInfo.IsDir() {
		return "", fmt.Errorf("passed '%s' path is not directory", dir)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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

// UpdateValuesImageTag updates values.yaml image tag field
func UpdateValuesImageTag(path, tag string) error {
	input, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	r := regexp.MustCompile(`tag: "[^"]+"`)
	output := r.ReplaceAll(input, []byte(fmt.Sprintf(`tag: "%s"`, tag)))

	return os.WriteFile(path, output, 0644)
}
