#!/usr/bin/env bash

path="$(realpath "${1}")"

docker run \
    -it \
    --rm \
    -v "${path}:${path}" \
    aksnestedworkspace:latest \
    az confcom katapolicygen -y "${path}"
