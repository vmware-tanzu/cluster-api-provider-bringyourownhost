#!/bin/bash

#this is a proto-bundle builder script
#the real one will be a separate stand alone executable
#the purpose of the script is simply to assemble all the bits of a bundle (archive configs, download packages, etc.)
#without creating OCI image

confPath=$1 #set default value
#packagesUrls=$2 #set default value(s)
#bundlePayloadPath=$N #set default value

#use absolute paths instead of changing the working dir

echo "Bundle configuration..."
if [ -d "$confPath" ]
then
	curPath=$(pwd)
	cd $confPath
	tar -cvf conf.tar etc
	mv ./conf.tar $curPath
fi

echo "Downloading K8s host components..."
#download to $bundlePayloadPath
#make sure all packages are renamed to generic names
