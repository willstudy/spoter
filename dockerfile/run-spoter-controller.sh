#!/bin/bash

if [ -z ${DEBUG_LEVEL} ]; then
    DEBUG_LEVEL=5
fi

cmd="/home/spoter/spoter-controller server -l ${DEBUG_LEVEL}
    --configFile "/home/spoter/config.json"
"
echo "cmd: " ${cmd}
eval ${cmd}
