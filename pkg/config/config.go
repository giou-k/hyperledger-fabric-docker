package config

import (
	"errors"
	"gopkg.in/yaml.v2"
	"os"
)

type (
	Config struct {
		HfToolPath    string         `yaml:"hfToolPath,omitempty"` // FIXME if exported in $PATH, leave it empty.
		ChannelName   string         `yaml:"channelName"`
		ConsensusType string         `yaml:"consensusType"`
		Orgs          []Organization `yaml:"orgs"`
	}

	Organization struct {
		Name     string     `yaml:name`
		Peers    []Peers    `yaml:peers`
		Orderers []Orderers `yaml:orderers`
	}

	Peers struct {
		Name string `yaml:name`
	}

	Orderers struct {
		Name string `yaml:name`
	}
)

// ParseConfig reads the config.yaml file, which is the configuration of our network.
func ParseConfig() (Config, error) {
	var c Config

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

	// Number of peers must be odd, so that the environmental variable of peer containers
	// "CORE_PEER_GOSSIP_BOOTSTRAP" can be calculated autonomous.
	for _, org := range c.Orgs {
		if len(org.Peers)/2 != 1 {
			return c, errors.New("The number of peers must be odd. ")
		}
	}

	return c, nil
}
