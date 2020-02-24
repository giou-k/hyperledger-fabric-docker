package main

import (
	"github.com/giou-k/hyperledger-fabric-docker/docker"
	"log"
	"os"
	"os/exec"
)

func main() {

	// Generate the crypto files for docker containers.
	cryptogen := exec.Command("/home/giou/go/src/github.com/fabric/fabric-samples/bin/cryptogen",
		"generate", "--config=./configs/crypto-config.yaml")

	err := cryptogen.Run()
	if err != nil {
		log.Printf("Cryptogen finished with error: %v", err)
		os.Exit(1)
	}

	// todo parse config from a config file.
	c := docker.Config{}

	cli, err := docker.NewClient()
	if err != nil {
		panic(err)
	}
	c = docker.Config{
		Peer:     []string{
			"peer0.org1.example.com",
			"peer1.org1.example.com",
		},
		Orderer:  []string{
			"orderer0.org1.example.com",
		},
		MyClient: cli,
	}

	err = c.CreateNetwork()
	if err != nil {
		log.Printf("CreateNetwork finished with error: %v", err)
		os.Exit(1)
	}
}
