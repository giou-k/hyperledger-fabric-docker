package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"log"
)

type CliInterface interface {
	CreateNetwork() error
	BringUp() error
	List() error
}

type Config struct {
	Peer    []string
	Orderer []string

	MyClient *client.Client
}

func (c *Config) CreateNetwork() error {
	var err error

	err = c.BringUp()
	if err != nil {
		return nil
	}

	err = c.List()
	if err != nil {
		return nil
	}

	return nil
}

func NewClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return cli, err
	}

	return cli, err
}

func (c *Config) BringUp() error {
	ctx := context.Background()

	config := &container.Config{
		Hostname:   "peer0.org1.example.com",
		Domainname: "peer0.org1.example.com",
		Env: []string{
			"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=giou", // fixme
			"CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock",
			"CORE_PEER_GOSSIP_USELEADERELECTION=true",
			"CORE_PEER_GOSSIP_ORGLEADER=false",
			"CORE_PEER_PROFILE_ENABLED=true",

			"CORE_PEER_ID=peer0.org1.example.com",
			"CORE_PEER_ADDRESS=peer0.org1.example.com:7051",
			"CORE_PEER_LISTENADDRESS=0.0.0.0:7051",
			"CORE_PEER_CHAINCODEADDRESS=peer0.org1.example.com:7052",
			"CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052",
			"CORE_PEER_GOSSIP_BOOTSTRAP=peer1.org1.example.com:8051",
			"CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.org1.example.com:7051",
			"CORE_PEER_LOCALMSPID=Org1MSP",
		},
		Cmd:   []string{"peer", "node", "start"},
		Image: "hyperledger/fabric-peer:1.4.4",
		Volumes: map[string]struct{}{
			"/var/run/:/host/var/run/": {},
			"crypto-config/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/msp:" +
				"/etc/hyperledger/fabric/msp": {},
			"peer0.org1.example.com:/var/hyperledger/production": {},
		},
		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric/peer",
		NetworkDisabled: false,
	}

	resp, err := c.MyClient.ContainerCreate(ctx, config, nil, nil,
		"peer0.org1.example.com")
	if err != nil {
		return err
	}
	log.Println("containerCreate resp: ", resp)

	if err := c.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return err
}

func (c *Config) List() error {
	containers, err := c.MyClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		log.Println(container.ID)
	}

	return nil
}
