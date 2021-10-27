#!/bin/bash

set -e

echo Pushing bundle "$*"

imgpkg push -f . -b $@

echo Done
