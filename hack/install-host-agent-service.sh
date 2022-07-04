#!/usr/bin/env bash

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# TODO: check if binary exists before installing, make the binary and kubeconfig path configurable 
make host-agent-binaries

cat << EOF > /etc/systemd/system/host-agent.service
[Unit]
Description=host-agent service
After=network.target

[Service]
Type=simple
WorkingDirectory=/home/kokoni/go/src/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/bin
ExecStart=/home/kokoni/go/src/github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/bin/byoh-hostagent-linux-amd64
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF


cat << EOF > /etc/systemd/system/host-agent-watcher.service
[Unit]
Description=host-agent restarter
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart host-agent.service

[Install]
WantedBy=multi-user.target
EOF

cat << EOF > /etc/systemd/system/host-agent-watcher.path
[Path]
Unit=host-agent-watcher.service
PathChanged=/root/.byoh/config


[Install]
WantedBy=multi-user.target
EOF

systemctl enable host-agent-watcher.{path,service}
systemctl start host-agent-watcher.{path,service}
systemctl enable host-agent.service
systemctl start host-agent.service
