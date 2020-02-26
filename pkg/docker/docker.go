package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/giou-k/hyperledger-fabric-docker/pkg/config"
	"log"
	"strings"
)

type Service struct {
	Cfg      *config.Config
	MyClient *client.Client
}

type CliInterface interface {
	CreateNetwork() error
	RunPeer() error
	List() error
}

// NewClient creates a docker client.
func NewClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return cli, err
	}

	return cli, err
}

// CreateNetwork creates the containers/nodes of our blockchain network.
func (s *Service) CreateNetwork() error {
	var err error

	// Create docker client.
	cli, err := NewClient()
	if err != nil {
		return err
	}
	s.MyClient = cli

	// Loop through organizations and run peer containers.
	for _, org := range s.Cfg.Orgs {
		for i, _ := range org.Peers {
			if err = s.RunPeer(org.Name, org.Peers, len(org.Peers), i); err != nil {
				return err
			}
		}
	}

	err = s.List()
	if err != nil {
		return err
	}

	return nil
}

// RunPeer runs peer containers.
func (s *Service) RunPeer(orgName string, peer []config.Peers, peerNum int, i int) error {
	ctx := context.Background()

	cfg := &container.Config{
		Hostname:   peer[i].Name,
		Domainname: peer[i].Name,
		Env: []string{
			"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=giou", // FIXME
			"CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock",
			"CORE_PEER_GOSSIP_USELEADERELECTION=true",
			"CORE_PEER_GOSSIP_ORGLEADER=false",
			"CORE_PEER_PROFILE_ENABLED=true",

			"CORE_PEER_ID=" + peer[i].Name,
			"CORE_PEER_ADDRESS=" + peer[i].Name + ":7051",
			"CORE_PEER_LISTENADDRESS=0.0.0.0:7051",
			"CORE_PEER_CHAINCODEADDRESS=" + peer[i].Name + ":7052",
			"CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052",
			"CORE_PEER_GOSSIP_BOOTSTRAP=" + peer[peerNum-(i+1)].Name,
			"CORE_PEER_GOSSIP_EXTERNALENDPOINT=" + peer[i].Name + ":7051",
			"CORE_PEER_LOCALMSPID=" + strings.Title(orgName) + "MSP",
		},
		Cmd:   []string{"peer", "node", "start"},
		Image: "hyperledger/fabric-peer:1.4.6",
		Volumes: map[string]struct{}{
			"/var/run/:/host/var/run/": {},
			"crypto-config/peerOrganizations/org1.example.com/peers/" + peer[i].Name + "/msp:" +
				"/etc/hyperledger/fabric/msp": {},
			peer[i].Name + ":/var/hyperledger/production": {},
		},
		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric/peer",
		NetworkDisabled: false,
	}

	resp, err := s.MyClient.ContainerCreate(ctx, cfg, nil, nil,
		peer[i].Name)
	if err != nil {
		return err
	}
	log.Println("containerCreate resp: ", resp)

	if err := s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return err
}

// RunOrderer runs orderer containers.
//func (s *Service) RunOrderer(orderer []config.Orderers, ordererNum int, i int) error {
//	ctx := context.Background()
//
//	cfg := &container.Config{
//		Hostname:   orderer[i].Name,
//		Domainname: orderer[i].Name,
//		Env: []string{
//			"FABRIC_LOGGING_SPEC=INFO",
//			"ORDERER_GENERAL_LISTENADDRESS=0.0.0.0",
//			"ORDERER_GENERAL_GENESISMETHOD=file",
//			"ORDERER_GENERAL_GENESISFILE=/var/hyperledger/orderer/orderer.genesis.block",
//			"ORDERER_GENERAL_LOCALMSPID=OrdererMSP",
//			"ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp", // FIXME
//		},
//		Cmd:   []string{"orderer"},
//		Image: "hyperledger/fabric-orderer:1.4.6",
//		Volumes: map[string]struct{}{
//			"channel-artifacts/genesis.block:/var/hyperledger/orderer/orderer.genesis.block": {},
//			"crypto-config/ordererOrganizations/example.com/orderers/" + orderer[i].Name + "/msp:" +
//				"/var/hyperledger/orderer/msp": {},
//			orderer[i].Name + ":/var/hyperledger/production/orderer": {},
//		},
//		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric",
//		NetworkDisabled: false,
//	}
//
//	resp, err := s.MyClient.ContainerCreate(ctx, cfg, nil, nil,
//		orderer[i].Name)
//	if err != nil {
//		return err
//	}
//	log.Println("containerCreate resp: ", resp)
//
//	if err := s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
//		return err
//	}
//
//	return err
//}

// List prints out the list of running containers.
func (s *Service) List() error {
	containers, err := s.MyClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}

	log.Println("List of containers that are running: ")
	for _, container := range containers {
		log.Println("container ID:", container.ID, "with container Name:", container.Names[0])
	}

	return nil
}
