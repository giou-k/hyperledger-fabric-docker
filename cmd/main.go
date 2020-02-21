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

	docker.BringUp()
}