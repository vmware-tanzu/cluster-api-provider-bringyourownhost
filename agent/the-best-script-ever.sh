set -x

export HOST_NAME=host2

docker stop $HOST_NAME
docker rm $HOST_NAME

cd ~/cluster-api-provider-byoh/agent

go build .

docker run --detach --tty --hostname $HOST_NAME --name $HOST_NAME --privileged --security-opt seccomp=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /lib/modules:/lib/modules:ro --network kind kindest/node:v1.19.11

docker cp agent $HOST_NAME:/agent

cp $KCONFIG /tmp/mgmt.conf

export KIND_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$(kind get clusters | grep test)-control-plane")
sed -i 's/    server\:.*/    server\: https\:\/\/'"$KIND_IP"'\:6443/g' /tmp/mgmt.conf

docker cp /tmp/mgmt.conf $HOST_NAME:/management-cluster.conf

docker exec -it $HOST_NAME bin/bash
