package config

import (
	"gopkg.in/yaml.v2"
	"os"
)

type Config struct {
	CryptogenPath string   `yaml:"cryptogenPath,omitempty"`
	Peers         []string `yaml:"peers"`
	Orderers      []string `yaml:"orderers"`
}

// ParseConfig reads the config.yaml file, which is the configuration of our network.
func ParseConfig() (Config, error) {
	var (
		err error
		c   Config
	)

	f, err := os.Open("./pkg/config/config.yaml")
	if err != nil {
		return c, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&c)
	if err != nil {
		return c, err
	}

	return c, nil
}
