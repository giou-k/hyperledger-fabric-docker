# Title

Bring up a blockchain network for Hyperledger Fabric using docker client. 

The network that is currently implemented is inspired by fabric-samples/first-nework build.

It includes:

* 5 orderer nodes.
* 4 peer nodes.
* 2 orgs for peers and 1 org for orderers.
* Raft consensus algorithm.
* TLS connection for orderers.

## Install

```
git clone https://github.com/giou-k/hyperledger-fabric-docker.git
```

## Usage
#### Compile and Run.
`cd hyperledger-fabric-docker`

`go build cmd/main.go`

`go run cmd/main`

#### Check network.
`docker ps -a`

`docker logs <container name>`

#### Kill network and Clear project dirs.
`./scripts/clear.sh`

and run manualy from terminal

`sudo rm -rf pkg/config/peer* pkg/config/orderer*`

We need sudo because the files are created inside the containers with root.

## Contributing

PRs accepted.

## License

MIT © giou-k

