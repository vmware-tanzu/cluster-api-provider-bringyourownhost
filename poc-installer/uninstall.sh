#!/bin/bash

#######################################
# DATA SECTION (global vars & consts) #
#######################################
readonly BAK_DIR="/tmp/byoh/bak"

###############
# ENTRY POINT #
###############

#stop the containerd daemon and uninstall it
sudo systemctl stop containerd
sudo systemctl disable containerd
sudo systemctl daemon-reload

#remove installed deb packages and their dependencies
apt remove -y kubeadm kubelet kubectl kubernetes-cni cri-tools
apt autoremove -y

#remove fs objects that we've installed (if any)
if [ -f "$BAK_DIR/remove.list" ]; then
    while IFS="" read -r line; do
        echo "Removing: $line"
        rm -rf "$line" 2>/dev/null
    done < $BAK_DIR/remove.list
fi

#restore fs objects that we've replaced (if any)
#TODO: add md5 hash check to verify the versions
# if the current ver. is different than the one we've installed
# do not remove it!
if [ -f "$BAK_DIR/restore.list" ]; then
    while IFS="" read -r line; do
        echo "Restoring: $line"
        cp -r "$line" 2>/dev/null
    done < $BAK_DIR/restore.list
fi

#cleanup all leftovers (backups and uninstall info)
rm -rf /tmp/byoh 2>/dev/null

#turn on swap
swapon -a

#turn on firewall
ufw enable
