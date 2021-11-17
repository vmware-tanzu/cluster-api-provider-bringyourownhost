#~/bin/bash

set -e

TAG=byoh-installer-e2e-test
IMG_INGR=byoh-ingredients-deb:$TAG
IMG_BLD=byoh-build-push-bundle:$TAG
IMG_REG=registry:2.7.1
CONT_REG=$TAG-registry
NET=$TAG-network

function cleanup()
{
    set +e

    ARG=$?
    echo "==Clean up"

    docker stop $CONT_REG

    docker rmi $IMG_INGR
    docker rmi $IMG_BLD
    docker rmi $IMG_REG

    # Clean up ingredients dir
    rm -rf $DIR_ING

    docker network rm $NET

    vagrant destroy -f

    popd

    exit $ARG
}

trap cleanup EXIT

echo "===Build BYOH Bundle Builder"
(cd agent/installer/bundle_builder/ && docker build -t $IMG_BLD .)
(cd agent/installer/bundle_builder/ingredients/deb/ && docker build -t $IMG_INGR .)

docker network create $NET

echo "===Start local bundle repo"
BUNDLE_REPO_PORT_HOST=5000
docker run -d -p $BUNDLE_REPO_PORT_HOST:5000 --rm --name $CONT_REG  --net $NET $IMG_REG
# Lookup by host name yields "http: server gave HTTP response to HTTPS client"
BUNDLE_REPO=`docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $CONT_REG`:$BUNDLE_REPO_PORT_HOST

echo "===Download bundle ingredients"
DIR_ING=$PWD/$TAG-ingredients
mkdir -p $DIR_ING && docker run --rm -v $DIR_ING:/ingredients $IMG_INGR

K8SVER=v1.22.3

echo "===Build and publish bundle"
docker run --rm -v $DIR_ING:/ingredients --net $NET --env BUILD_ONLY=0 $IMG_BLD $BUNDLE_REPO/byoh/byoh-bundle-ubuntu_20.04.1_x86-64_k8s_$K8SVER

pushd agent/installer/e2e
go build ../cli
echo "===Spin up test vm"
vagrant up
echo "===Install bundle inside vm"
vagrant ssh -c "cd /vagrant && sudo ./test.sh $BUNDLE_REPO_PORT_HOST $K8SVER"
