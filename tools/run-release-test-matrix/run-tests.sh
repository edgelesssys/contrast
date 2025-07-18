#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

# Create an associative array to hold platform.name -> test list
declare -A platform_tests

# This should point to the workflows called in release.yml, excluding the release test.
declare -A files=(
  [".github/workflows/e2e_nightly.yml"]="jobs.test_matrix.strategy.matrix"
  [".github/workflows/e2e_regression.yml"]="jobs.regression-test.strategy.matrix"
)

for file in "${!files[@]}"; do
  MATRIX_FILE="$file"
  MATRIX_PATH="${files[$file]}"

  echo "Processing $MATRIX_FILE at path $MATRIX_PATH" >&2

  # Read all test names
  mapfile -t tests < <(yq ".$MATRIX_PATH.test-name[]" "$MATRIX_FILE")

  # Read number of platforms
  platform_count=$(yq ".$MATRIX_PATH.platform | length" "$MATRIX_FILE")

  for ((i = 0; i < platform_count; i++)); do
    name=$(yq ".$MATRIX_PATH.platform[$i].name" "$MATRIX_FILE")
    self_hosted=$(yq ".$MATRIX_PATH.platform[$i].self-hosted" "$MATRIX_FILE")

    valid_tests=()
    for test in "${tests[@]}"; do
      if ! yq -o=json ".$MATRIX_PATH.exclude[]?" "$MATRIX_FILE" |
        jq -e --argjson sh "$self_hosted" --arg t "$test" 'select(."test-name" == $t and .platform."self-hosted" == $sh)' >/dev/null; then
        valid_tests+=("$test")
      fi
    done

    # Append valid tests to platform
    for test in "${valid_tests[@]}"; do
      if [[ -z ${platform_tests[$name]+set} ]] || [[ ! " ${platform_tests[$name]} " =~ $test ]]; then
        platform_tests[$name]+=" $test"
      fi
    done
  done

  # Apply includes
  include_count=$(yq ".$MATRIX_PATH.include | length" "$MATRIX_FILE" 2>/dev/null || echo 0)
  if [[ $include_count -gt 0 ]]; then
    for ((j = 0; j < include_count; j++)); do
      name=$(yq ".$MATRIX_PATH.include[$j].platform.name" "$MATRIX_FILE")
      test=$(yq ".$MATRIX_PATH.include[$j].test-name" "$MATRIX_FILE")
      if [[ -z ${platform_tests[$name]+set} ]] || [[ ! " ${platform_tests[$name]} " =~ $test ]]; then
        platform_tests[$name]+=" $test"
      fi
    done
  fi
done

# Output merged results
echo "Discovered the following test matrix:" >&2
for platform in "${!platform_tests[@]}"; do
  echo "$platform:${platform_tests[$platform]}" >&2
done

# Run tests
for platform in "${!platform_tests[@]}"; do
  echo "Setting default_platform to $platform in justfile.env" >&2
  sed -i "s/^default_platform=.*/default_platform=\"$platform\"/" justfile.env
  echo "Getting credentials.." >&2
  just get-credentials
  for test in ${platform_tests[$platform]}; do
    echo "Running test $test on platform $platform" >&2
    just e2e "$test"
  done
done
