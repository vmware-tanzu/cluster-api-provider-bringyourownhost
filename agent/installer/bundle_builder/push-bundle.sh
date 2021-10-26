#!/bin/bash

set -e

echo Pushing bundle "$*"

mkdir .imgpkg
echo kbld
kbld --imgpkg-lock-output .imgpkg/images.yml
imgpkg push -f . -b $@

echo Done
