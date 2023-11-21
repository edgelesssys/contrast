#!/usr/bin/env bash

set -euo pipefail

repoRoot="$(git rev-parse --show-toplevel)"
echo "repoRoot: ${repoRoot}"

yamlPaths=("$@")

flags=()
flags+=("-it")
flags+=("--rm")
flags+=("--env=RUST_LOG=info")
flags+=("-v=${repoRoot}/tools/genpolicy-own.json:/workspace/genpolicy-policy.json")
flags+=("-v=${repoRoot}/tools/rules.rego:/workspace/rules.rego")
flags+=("-v=${repoRoot}/tools/genpolicy.cache:/workspace/layers_cache")

commands=()

for yamlPath in "${yamlPaths[@]}"; do
    realYamlPath="$(realpath "${yamlPath}")"
    echo "yaml path: ${realYamlPath}"
    flags+=("-v=${realYamlPath}:/workspace/${yamlPath}")
    commands+=("genpolicy -u -j genpolicy-policy.json -y /workspace/${yamlPath} &&")
done

commands+=("echo done")

docker run "${flags[@]}" genpolicy:latest bash -c "${commands[*]}"
