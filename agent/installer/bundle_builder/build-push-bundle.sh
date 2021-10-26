#!/bin/bash

set -e

build-payload.sh $1 $2
push-bundle.sh ${@:3}

