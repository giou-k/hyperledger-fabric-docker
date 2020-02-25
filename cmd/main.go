package main

import (
	"github.com/giou-k/hyperledger-fabric-docker/pkg/config"
	"github.com/giou-k/hyperledger-fabric-docker/pkg/docker"
	"log"
	"os"
	"os/exec"
)

func main() {

	// Parse the configuration of our network.
	cfg, err := config.ParseConfig()
	if err != nil {
		log.Printf("ParseConfig finished with error: %v", err)
		os.Exit(1)
	}
	s := &docker.Service{
		Cfg: &cfg,
	}

	// Generate the crypto files for docker containers.
	cryptogen := exec.Command(s.Cfg.CryptogenPath,
		"generate", "--config=./pkg/config/crypto-config.yaml")

	err = cryptogen.Run()
	if err != nil {
		log.Printf("Cryptogen finished with error: %v", err)
		os.Exit(1)
	}

	// Run the containers.
	err = s.CreateNetwork()
	if err != nil {
		log.Printf("CreateNetwork finished with error: %v", err)
		os.Exit(1)
	}
}
