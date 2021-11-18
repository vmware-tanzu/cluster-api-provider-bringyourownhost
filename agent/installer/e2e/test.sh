#~/bin/bash

set -e

TAG=byoh-installer-e2e-test
export REPOPORT=5005
export K8SVER="v1.22.3"

function cleanup()
{
    set +e

    ARG=$?
    echo "==Clean up"

    docker-compose -p $TAG -f build.yml -f core.yml down --rmi all --volumes

    vagrant destroy -f

    rm cli
    rm -rf $K8SVER*

    popd

    exit $ARG
}

trap cleanup EXIT

pushd agent/installer/e2e

echo "===Download bundle ingredients"
docker-compose -p $TAG -f core.yml -f build.yml up --build ingredients-deb

echo "===Starting bundle repository"
docker-compose -p $TAG -f core.yml -f build.yml up --build bundle-repo.local &

echo "===Build and publish bundle"
docker-compose -p $TAG -f core.yml -f build.yml up --build bundle-builder

echo "===Build cli"
go build ../cli
echo "===Spin up test vm"
vagrant up
echo "===Install bundle inside vm"
vagrant ssh -c "cd /vagrant && sudo ./test-in-vm.sh $REPOPORT $K8SVER"
