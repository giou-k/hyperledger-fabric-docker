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
	"sync"
)

type Service struct {
	MyClient *client.Client

	Cfg *config.Config
}

type NetworkAPI interface {
	CreateNetwork() error
	RunPeer(orgName string, peer []config.Peers, projectPath string, i int,
		errChanPeer chan error, wgPeerDone chan bool)
	RunOrderer(orderer []config.Orderers, projectPath string, i int,
		errChanOrderer chan error, wgOrdererDone chan bool)
	List() error
}

// NewClient creates a docker client.
func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// CreateNetwork creates the containers/nodes of our blockchain network.
func (s *Service) CreateNetwork() error {
	// Create docker client.
	var (
		cli *client.Client

		err            error
		errChanPeer    = make(chan error, 1)
		errChanOrderer = make(chan error, 1)

		wg            sync.WaitGroup
		wgPeerDone    = make(chan bool) // todo change this to ctx.Done()
		wgOrdererDone = make(chan bool) // todo change this to ctx.Done()
	)

	if cli, err = NewClient(); err != nil {
		return errors.Wrap(err, "NewClient failed with error")
	}
	s.MyClient = cli

	ctx := context.TODO()
	respNet, err := cli.NetworkCreate(ctx, "giou_net", types.NetworkCreate{})
	if err != nil {
		return errors.Wrap(err, "NetworkCreate failed with error")
	}
	log.Println("Network has been created wth ID: ", respNet.ID)

	// In need of absolute path to bind/mount host:container paths.
	projectPath, err := filepath.Abs("./")
	if err != nil {
		return errors.Wrap(err, "Failed to get project's path with error")
	}

	wg.Add(1)

	go func(orgs []config.Organization) {
		defer wg.Done()

		for _, org := range orgs {
			for i := range org.Peers {

				go s.RunPeer(org.Name, org.Peers, projectPath, i, errChanPeer, wgPeerDone)

				select {
				case <-wgPeerDone:
					log.Println("carry on...")

					break
				case err := <-errChanPeer:
					close(errChanPeer)
					log.Fatal("Error: ", err)
					//return err
					return
				}
			}
		}
	}(s.Cfg.Orgs[1:])

	wg.Add(1)

	go func(org config.Organization) {
		defer wg.Done()

		for i := range org.Orderers {

			go s.RunOrderer(org.Orderers, projectPath, i, errChanOrderer, wgOrdererDone)

			select {
			case <-wgOrdererDone:
				// carry on
				log.Println("carry on orderer...")

				break
			case err := <-errChanOrderer:
				close(errChanOrderer)
				log.Fatal("Error Orderer: ", err)
				//return err
				return
			}
		}
	}(s.Cfg.Orgs[0])

	// waits until containers are up and running
	wg.Wait()

	return s.List()
}

// RunPeer runs peer containers.
func (s *Service) RunPeer(orgName string, peer []config.Peers, projectPath string, i int,
	errChanPeer chan error, wgPeerDone chan bool) {
	ctx := context.TODO()
	cfg, hostConfig := configPeer(peer, projectPath, orgName, i)

	resp, err := s.MyClient.ContainerCreate(ctx, cfg, hostConfig, nil,
		peer[i].Name)
	if err != nil {
		errChanPeer <- errors.Wrap(err, "ContainerCreate failed with error")
		return
	}

	log.Println("ContainerCreate for peer response: ", resp)

	err = s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		errChanPeer <- errors.Wrap(err, "ContainerStart failed with error")
		return
	}

	log.Println("ContainerStart for peer succeed.")

	wgPeerDone <- true
}

// RunOrderer runs orderer containers.
func (s *Service) RunOrderer(orderer []config.Orderers, projectPath string, i int,
	errChanOrderer chan error, wgOrdererDone chan bool) {
	ctx := context.TODO()

	cfg, hostConfig, err := configOrderer(orderer[i], projectPath)
	if err != nil {
		errChanOrderer <- errors.Wrap(err, "configOrderer failed with error")
		return
	}

	resp, err := s.MyClient.ContainerCreate(ctx, cfg, hostConfig, nil, orderer[i].Name)
	if err != nil {
		errChanOrderer <- errors.Wrap(err, "ContainerCreate failed with error")
		return
	}

	log.Println("ContainerCreate for orderer response: ", resp)

	//hijackResp, err := s.MyClient.ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{})
	//defer hijackResp.Close()

	if err := s.MyClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		errChanOrderer <- errors.Wrap(err, "ContainerStart failed with error")
		return
	}

	log.Println("ContainerStart for orderer succeed.")

	wgOrdererDone <- true
}

// List prints out the list of running containers.
func (s *Service) List() error {
	containers, err := s.MyClient.ContainerList(context.TODO(), types.ContainerListOptions{})
	if err != nil {
		return errors.Wrap(err, "ContainerList failed with error")
	}

	log.Println("List of containers that are running: ")

	for _, thisContainer := range containers {
		log.Println("container ID:", thisContainer.ID, "with container Name:", thisContainer.Names[0])
	}

	return nil
}

// Returns the environment variables in KEY="VALUE" form.
func envVars(peer []config.Peers, i int, orgName string) []string {
	switch peer {
	case nil:
		return []string{
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
		}
	default:
		return []string{
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
			"CORE_PEER_GOSSIP_BOOTSTRAP=" + peer[len(peer)-(i+1)].Name,
			"CORE_PEER_GOSSIP_EXTERNALENDPOINT=" + peer[i].Name + ":7051",
			"CORE_PEER_LOCALMSPID=" + strings.Title(orgName) + "MSP",
		}
	}
}

// configPeer configures docker variables for each peer.
func configPeer(peer []config.Peers, projectPath string, orgName string, i int) (*container.Config, *container.HostConfig) {
	cfg := &container.Config{
		Hostname:        peer[i].Name,
		Domainname:      peer[i].Name,
		Env:             envVars(peer, i, orgName),
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

	return cfg, hostConfig
}

// configOrderer configures docker variables for each orderer.
func configOrderer(orderer config.Orderers, projectPath string) (*container.Config, *container.HostConfig, error) {
	cfg := &container.Config{
		Hostname:        orderer.Name,
		Domainname:      orderer.Name,
		Env:             envVars(nil, 0, ""),
		Cmd:             []string{"orderer"},
		Image:           "hyperledger/fabric-orderer:1.4.6",
		WorkingDir:      "/opt/gopath/src/github.com/hyperledger/fabric",
		NetworkDisabled: false,
	}

	containerPort, err := nat.NewPort("tcp", "7050")
	if err != nil {
		return nil, nil, errors.Wrap(err, "NewPort failed with error")
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			projectPath + "/pkg/config/crypto-config/ordererOrganizations/example.com/orderers/" +
				orderer.Name + "/msp:" + "/var/hyperledger/orderer/msp",
			projectPath + "/pkg/config/crypto-config/ordererOrganizations/example.com/orderers/" +
				orderer.Name + "/tls:" + "/var/hyperledger/orderer/tls",
			projectPath + "/pkg/config/" + orderer.Name +
				":/var/hyperledger/production/orderer",
			projectPath + "/pkg/config/channel-artifacts/genesis.block:/var/hyperledger/orderer/orderer.genesis.block",
		},
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{{
				HostIP:   "0.0.0.0",
				HostPort: orderer.Port,
			},
			},
		},
		NetworkMode: "giou_net",
	}

	return cfg, hostConfig, nil
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
