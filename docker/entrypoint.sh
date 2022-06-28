#!/bin/sh

set -e
user=ipfs
nid=hostname | cut -d- -f3
export CLUSTER_PEERNAME=`hostname`
export CLUSTER_SECRET=""
export CLUSTER_IPFSHTTP_NODEMULTIADDRESS=/dns4/go-ipfs-${nid}.go-ipfs-all.ipfs.svc.cluster.local/tcp/5001
export CLUSTER_CRDT_TRUSTEDPEERS="*"
export CLUSTER_RESTAPI_HTTPLISTENMULTIADDRESS="/ip4/0.0.0.0/tcp/9094"
export CLUSTER_MONITORPINGINTERVAL="2s"

if [ -n "$DOCKER_DEBUG" ]; then
   set -x
fi

if [ `id -u` -eq 0 ]; then
    echo "Changing user to $user"
    # ensure directories are writable
    su-exec "$user" test -w "${IPFS_CLUSTER_PATH}" || chown -R -- "$user" "${IPFS_CLUSTER_PATH}"
    exec su-exec "$user" "$0" $@
fi

# Only ipfs user can get here
ipfs-cluster-service --version

if [ -e "${IPFS_CLUSTER_PATH}/service.json" ]; then
    echo "Found IPFS cluster configuration at ${IPFS_CLUSTER_PATH}"
else
    echo "This container only runs ipfs-cluster-service. ipfs needs to be run separately!"
    echo "Initializing default configuration..."
    ipfs-cluster-service init --consensus "${IPFS_CLUSTER_CONSENSUS}"
fi

exec ipfs-cluster-service $@
