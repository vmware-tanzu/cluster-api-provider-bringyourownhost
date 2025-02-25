#!/bin/bash

# Copyright 2021 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0


INGREDIENTS_PATH=$1
CONFIG_PATH=$2

set -e

echo Building bundle...

echo Ingredients $INGREDIENTS_PATH
ls -l $INGREDIENTS_PATH

env

echo Strip version to well-known names
# Mandatory
mv $INGREDIENTS_PATH/*containerd* containerd.tar
mv $INGREDIENTS_PATH/*kubeadm*.deb ./kubeadm.deb
mv $INGREDIENTS_PATH/*kubelet*.deb ./kubelet.deb
mv $INGREDIENTS_PATH/*kubectl*.deb ./kubectl.deb
# Optional
mv  $INGREDIENTS_PATH/*cri-tools*.deb cri-tools.deb > /dev/null | true
mv  $INGREDIENTS_PATH/*kubernetes-cni*.deb kubernetes-cni.deb > /dev/null | true

echo Configuration $CONFIG_PATH
ls -l $CONFIG_PATH

echo Add configuration under well-known name
(cd $CONFIG_PATH && tar -cvf conf.tar *)
cp $CONFIG_PATH/conf.tar .

echo Creating bundle tar
tar -cvf /bundle/bundle.tar *

echo Done
