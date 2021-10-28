#!/bin/bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -e

build-bundle.sh $1 $2
if [ $BUILD_ONLY -eq 0 ]
then
push-bundle.sh ${@:3}
fi

