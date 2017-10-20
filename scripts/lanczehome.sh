#!/usr/bin/env bash

if [[ ${BASH_SOURCE} == $0 ]] ; then
    echo "Error!!! Run command as: source lanczehome.sh"
    exit 1
else
    SCRIPTS_DIR=`dirname $(readlink -f ${BASH_SOURCE})`
    . ${SCRIPTS_DIR}/lanczevars.sh
    cd ${LANCZE_HOME}/src/lancze/server
    echo "Current directory changed to:"
    pwd
fi


