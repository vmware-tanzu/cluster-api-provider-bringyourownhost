#!/bin/bash

set -e

build-bundle.sh $1 $2
if [ $BUILD_ONLY -eq 0 ]
then
push-bundle.sh ${@:3}
fi

