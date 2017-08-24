package helper

import (
	"fmt"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// GetYamlConfig parse .yaml
func GetYamlConfig(path string, config interface{}) error {
	configContent, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("can't read config %q: %s", path, err)
	}

	if err = yaml.Unmarshal(configContent, config); err != nil {
		return fmt.Errorf("invalid yaml in config %q: %s", path, err)
	}

	return nil
}
