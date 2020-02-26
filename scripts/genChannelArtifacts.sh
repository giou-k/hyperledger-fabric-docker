#!/bin/bash
set -e

set -x
CHANNEL_NAME=$1
CONSENSUS_TYPE=$2
FABRIC_CFG_PATH=$3
HF_TOOL_PATH=$4
set +x

# system channel name defaults to "giou-sys-channel"
SYS_CHANNEL=giou-sys-channel

export FABRIC_CFG_PATH
export PATH=HF_TOOL_PATH:${PWD}:$PATH

# Generate all the certificates for msps, orderers and peers.
generateCertificates () {
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. exiting"
    exit 1
  fi

  echo
  echo "##########################################################"
  echo "##### Generate certificates using cryptogen tool #########"
  echo "##########################################################"

  if [ -d "./pkg/config/crypto-config" ]; then
    rm -Rf ./pkg/config/crypto-config
  fi
  set -x
  cryptogen generate --config=./pkg/config/crypto-config.yaml && mv -f crypto-config ./pkg/config
  res=$?
  set +x
  if [ ${res} -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  echo

}

# Generate orderer genesis block, channel configuration transaction and
# anchor peer update transactions
function generateChannelArtifacts() {
  which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. exiting"
    exit 1
  fi

  echo "##########################################################"
  echo "#########  Generating Orderer Genesis block ##############"
  echo "##########################################################"
  # Note: For some unknown reason (at least for now) the block file can't be
  # named orderer.genesis.block or the orderer will fail to launch!
  set -x
  if [ "${CONSENSUS_TYPE}" == "solo" ]; then
    configtxgen -profile TwoOrgsOrdererGenesis -channelID ${SYS_CHANNEL} -outputBlock ./pkg/config/genesis.block
  elif [ "${CONSENSUS_TYPE}" == "kafka" ]; then
    configtxgen -profile SampleDevModeKafka -channelID ${SYS_CHANNEL} -outputBlock ./pkg/config/genesis.block
  elif [ "${CONSENSUS_TYPE}" == "etcdraft" ]; then
    configtxgen -profile SampleMultiNodeEtcdRaft -channelID ${SYS_CHANNEL} -outputBlock ./pkg/config/genesis.block
  else
    set +x
    echo "unrecognized CONSESUS_TYPE='${CONSENSUS_TYPE}'. exiting"
    exit 1
  fi
  res=$?
  set +x

  if [ ${res} -ne 0 ]; then
    echo "Failed to generate orderer genesis block..."
    exit 1
  fi

  echo
  echo "#################################################################"
  echo "### Generating channel configuration transaction 'channel.tx' ###"
  echo "#################################################################"

  set -x
  configtxgen -profile TwoOrgsChannel -outputCreateChannelTx ./pkg/config/channel.tx -channelID ${CHANNEL_NAME}
  res=$?
  set +x

  if [ ${res} -ne 0 ]; then
    echo "Failed to generate channel configuration transaction..."
    exit 1
  fi

  echo
  echo "#################################################################"
  echo "#######    Generating anchor peer update for Org1MSP   ##########"
  echo "#################################################################"

  set -x
  configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./pkg/config/Org1MSPanchors.tx -channelID ${CHANNEL_NAME} \
  -asOrg Org1MSP
  res=$?
  set +x

  if [ ${res} -ne 0 ]; then
    echo "Failed to generate anchor peer update for Org1MSP..."
    exit 1
  fi

  echo
  echo "#################################################################"
  echo "#######    Generating anchor peer update for Org2MSP   ##########"
  echo "#################################################################"
  set -x

  configtxgen -profile TwoOrgsChannel -outputAnchorPeersUpdate ./pkg/config/Org2MSPanchors.tx -channelID ${CHANNEL_NAME} \
   -asOrg Org2MSP
  res=$?
  set +x

  if [ ${res} -ne 0 ]; then
    echo "Failed to generate anchor peer update for Org2MSP..."
    exit 1
  fi
  echo
}

generateCertificates

generateChannelArtifacts