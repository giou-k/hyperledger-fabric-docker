#!/bin/bash
set -e

# remove all containers
docker rm -f $(docker ps -aq)

# remove crypto-config dir
rm -rf ./pkg/config/crypto-config/*

# remove volumes
rm -rf pkg/config/orderer*
rm -rf pkg/config/peer*

#remove channel artifact
rm -rf pkg/config/channel.tx
rm -rf pkg/config/genesis.block
rm -rf pkg/config/Org*
