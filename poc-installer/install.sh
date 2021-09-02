#!/bin/bash

#######################################
# DATA SECTION (global vars & consts) #
#######################################
readonly WORK_DIR="/tmp/byoh"
readonly PAYLOAD_DIR="./payload"
readonly INSTALL_TMP_DIR="$WORK_DIR/root"

####################
# HELPER FUNCTIONS #
####################

recursiveExtractLayers()
{
    if [ ! -d "$INSTALL_TMP_DIR" ]
    then
        mkdir -p "$INSTALL_TMP_DIR"
    fi

    for layerTar in $(find $WORK_DIR -name "layer.tar")
    do
        echo "Extracting layer $layerTar..."
        tar -C "$INSTALL_TMP_DIR" -xf "$layerTar"
    done
}

###############
# ENTRY POINT #
###############

#prepare
if [ ! -d "$WORK_DIR" ]
then
    echo "*** Directory $WORK_DIR does not exists, creating backup file structure..."
    mkdir -p "$INSTALL_TMP_DIR"
fi


#extract all archives from the "payload" directory
#fsObj = file system object (file or dir, in our case just tarballs)
cd $PAYLOAD_DIR

for fsObj in *.tar
do 
    echo "Processing: $fsObj"
    ../tkgpkg_fsbackup.sh "$fsObj"
    tar -C "$INSTALL_TMP_DIR" -xvf "$fsObj"
done

recursiveExtractLayers
echo "All tarballs extracted."

#install all debs
while IFS="" read -r packageFile || [ -n "$packageFile" ]
do
    apt install -y "./$packageFile"
done < debinstall.list

#install all the content extracted from the tarballs
echo "Installing tarballs contents..."
cp -R $INSTALL_TMP_DIR/* /

#turn off firewall
ufw disable

#turn off swap
swapoff -a

#start containerd
systemctl daemon-reload
systemctl start containerd