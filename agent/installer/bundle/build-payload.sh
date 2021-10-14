#!/bin/bash

path=$1

if [ -z "$path" ]
then
	echo "You must specify absolute or relative path"
	echo "Example: build-payload.sh ./ubuntu/20_04/k8s/1_22/"
	exit
fi

if [ -d "$path" ]
then
	curPath=$(pwd)
	cd $path
	tar -cvf conf.tar etc
	mv ./conf.tar $curPath
fi
