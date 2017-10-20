#!/usr/bin/env bash

if [[ ${BASH_SOURCE[@]} == $0 ]] ; then
    echo "Error!!! Run command as: source lanczevars.sh"
    exit 1
fi

SCRIPTS_DIR=`dirname $(readlink -f ${BASH_SOURCE})`
export LANCZE_HOME=`dirname "$SCRIPTS_DIR"`
export GOPATH=${LANCZE_HOME}:${LANCZE_HOME}/vendor
echo "GOPATH=$GOPATH"
export CGO_ENABLED=0
