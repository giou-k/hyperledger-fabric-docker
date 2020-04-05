#!/bin/bash
set -e

# remove all containers and prune the network
docker rm -f $(docker ps -aq)
docker network prune -f

# remove crypto-config dir
rm -rf ./pkg/config/crypto-config/*

#remove channel artifact
rm -rf ./pkg/config/channel-artifacts/*