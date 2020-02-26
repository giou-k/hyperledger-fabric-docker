package main

import (
	"github.com/giou-k/hyperledger-fabric-docker/pkg/config"
	"github.com/giou-k/hyperledger-fabric-docker/pkg/docker"
	"log"
	"os"
	"os/exec"
)

const FABRIC_CFG_PATH = "./pkg/config"

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

	// Generate the genesis block for orderer channel and the anchor peers tx file for the common channel.
	output, err := exec.Command("./scripts/genChannelArtifacts.sh", s.Cfg.ChannelName, s.Cfg.ConsensusType,
		FABRIC_CFG_PATH, s.Cfg.HfToolPath).CombinedOutput()

	//err = configtxgen.Run()
	if err != nil {
		log.Println("Error when running genChannelArtifacts.sh.  Output:")
		log.Println(string(output))
		log.Printf("Got exit status: %s\n", err.Error())
		os.Exit(1)
	}

	// Run the containers.
	err = s.CreateNetwork()
	if err != nil {
		log.Printf("CreateNetwork finished with error: %v", err)
		os.Exit(1)
	}
}
