#~/bin/bash

#set -e

INSTALLER_ROOT=`realpath ..`
TAG=byoh-installer-e2e-test
IMG_INGR=byoh-ingredients-deb:$TAG
IMG_BLD=byoh-build-push-bundle:$TAG

echo "===Build BYOH Bundle Builder"
(cd agent/installer/bundle_builder/ && docker build -t $IMG_BLD .)
(cd agent/installer/bundle_builder/ingredients/deb/ && docker build -t $IMG_INGR .)

echo "===Start local bundle repo"
BUNDLE_REPO_PORT_HOST=5000
BUNDLE_REPO=10.26.226.219:$BUNDLE_REPO_PORT_HOST
#BUNDLE_REPO=https://localhost
docker run -d -p $BUNDLE_REPO_PORT_HOST:5000 --rm --name registry-$TAG registry:2.7.1

K8SVER=v1.22.3

echo "===Download bundle ingredients"
DIR_ING=$PWD/$TAG-ingredients
mkdir -p $DIR_ING && docker run --rm -v $DIR_ING:/ingredients $IMG_INGR

echo "===Build and publish bundle"
docker run --rm -v $DIR_ING:/ingredients --env BUILD_ONLY=0 $IMG_BLD $BUNDLE_REPO/byoh-bundle-ubuntu_20.04.1_x86-64_k8s_$K8SVER

# Clean up ingredients dir
rm -rf $DIR_ING

pushd agent/installer/e2e
go build ../cli
echo "===Spin up test vm"
vagrant up
echo "===Install bundle inside vm"
vagrant ssh -c "cd /vagrant && sudo ./test.sh $BUNDLE_REPO_PORT_HOST $K8SVER"
vagrant destroy -f
popd

docker stop registry-$TAG


