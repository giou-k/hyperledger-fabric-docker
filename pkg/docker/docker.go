package docker

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/giou-k/hyperledger-fabric-docker/pkg/config"
	"github.com/pkg/errors"
	"log"
	"path/filepath"
	"strings"
)

type Service struct {
	MyClient *client.Client

	Cfg *config.Config
}

type NetworkAPI interface {
	CreateNetwork() error
	RunPeer(orgName string, peer []config.Peers, peerNum int, projectPath string, i int) error
	RunOrderer(orderer []config.Orderers, projectPath string, i int) error
	List() error
}

// NewClient creates a docker client.
func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// CreateNetwork creates the containers/nodes of our blockchain network.
func (s *Service) CreateNetwork() error {
	var (
		cli *client.Client

		err error
	)

	// Create docker client.
	if cli, err = NewClient(); err != nil {
		return err
	}
	s.MyClient = cli

	ctx := context.Background() // Fixme: have the ctx as parameter in every func.
	respNet, err := cli.NetworkCreate(ctx, "giou_net", types.NetworkCreate{})
	if err != nil {
		return err
	}
	log.Println("network has been created wth ID: ", respNet.ID)

	// In need of absolute path to bind/mount host:container paths.
	projectPath, err := filepath.Abs("./")

	// Loop through organizations and run peer containers.
	for _, org := range s.Cfg.Orgs {
		for i := range org.Peers {
			if err = s.RunPeer(org.Name, org.Peers, len(org.Peers), projectPath, i); err != nil {
				return err
			}
		}
		for i := range org.Orderers {
			if err = s.RunOrderer(org.Orderers, projectPath, i); err != nil {
				return err
			}
		}
	}

	return s.List()
}

// RunPeer runs peer containers.
func (s Service) RunPeer(orgName string, peer []config.Peers, peerNum int, projectPath string, i int) error {
	ctx := context.Background()

	cfg := &container.Config{
		Hostname:   peer[i].Name,
		Domainname: peer[i].Name,
		Env: []string{
			"CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock",
			"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=giou_net",
			"FABRIC_LOGGING_SPEC=INFO",
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
		Cmd:             []string{"peer", "node", "start"},
		Image:           "hyperledger/fabric-peer:1.4.6",
		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric/peer",
		NetworkDisabled: false,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			"/var/run/:/host/var/run/",
			projectPath + "/pkg/config/crypto-config/peerOrganizations/" +
				orgName + ".example.com/peers/" + peer[i].Name + "/msp:" + "/etc/hyperledger/fabric/msp",
			projectPath + "/pkg/config/" + peer[i].Name +
				":/var/hyperledger/production",
		},
		NetworkMode: "giou_net",
	}

	resp, err := s.MyClient.ContainerCreate(ctx, cfg, hostConfig, nil,
		peer[i].Name)
	if err != nil {
		return err
	}
	log.Println("containerCreate resp: ", resp)

	return s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
}

// RunOrderer runs orderer containers.
func (s *Service) RunOrderer(orderer []config.Orderers, projectPath string, i int) error {
	ctx := context.Background()

	cfg := &container.Config{
		Hostname:   orderer[i].Name,
		Domainname: orderer[i].Name,
		Env: []string{
			"FABRIC_LOGGING_SPEC=INFO",
			"ORDERER_GENERAL_LISTENADDRESS=0.0.0.0",
			"ORDERER_GENERAL_GENESISMETHOD=file",
			"ORDERER_GENERAL_GENESISFILE=/var/hyperledger/orderer/orderer.genesis.block",
			"ORDERER_GENERAL_LOCALMSPID=OrdererMSP",
			"ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp", // FIXME
			//	TLS
			"ORDERER_GENERAL_TLS_ENABLED=true",
			"ORDERER_GENERAL_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key",
			"ORDERER_GENERAL_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt",
			"ORDERER_GENERAL_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]",
			"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE=/var/hyperledger/orderer/tls/server.crt",
			"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY=/var/hyperledger/orderer/tls/server.key",
			"ORDERER_GENERAL_CLUSTER_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]",
		},
		Cmd:             []string{"orderer"},
		Image:           "hyperledger/fabric-orderer:1.4.6",
		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric",
		NetworkDisabled: false,
	}

	containerPort, err := nat.NewPort("tcp", "7050")
	if err != nil {
		return errors.Wrap(err, "Unable to get the port")
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			projectPath + "/pkg/config/crypto-config/ordererOrganizations/example.com/orderers/" +
				orderer[i].Name + "/msp:" + "/var/hyperledger/orderer/msp",
			projectPath + "/pkg/config/crypto-config/ordererOrganizations/example.com/orderers/" +
				orderer[i].Name + "/tls:" + "/var/hyperledger/orderer/tls",
			projectPath + "/pkg/config/" + orderer[i].Name +
				":/var/hyperledger/production/orderer",
			projectPath + "/pkg/config/channel-artifacts/genesis.block:/var/hyperledger/orderer/orderer.genesis.block",
		},
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{
				HostIP:   "0.0.0.0",
				HostPort: orderer[i].Port,
			},
			},
		},
		NetworkMode: "giou_net",
	}

	resp, err := s.MyClient.ContainerCreate(ctx, cfg, hostConfig, nil, orderer[i].Name)
	if err != nil {
		return err
	}
	log.Println("containerCreate for orderers response: ", resp)

	//hijackResp, err := s.MyClient.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{})
	//defer hijackResp.Close()
	//// TODO: should I put defer in a goroutine?

	if err := s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	return err
}

// List prints out the list of running containers.
func (s *Service) List() error {
	containers, err := s.MyClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}

	log.Println("List of containers that are running: ")
	for _, thisContainer := range containers {
		log.Println("container ID:", thisContainer.ID, "with container Name:", thisContainer.Names[0])
	}

	return nil
}

//func printStream(streamer io.Reader) error {
//
//	var w io.Writer
//	if n, err := io.Copy(w, streamer); n == 0 || err != nil {
//		return err
//	}
//
//	var stream string
//	if n, err := io.WriteString(w, stream); n == 0 || err != nil {
//		return err
//	}
//	log.Println("hijackResp: ", stream)
//
//	return nil
//}
