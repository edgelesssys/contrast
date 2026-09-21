#!/usr/bin/env bash
# Copyright 2025 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

# Runs the tests that gate a release, as defined by the workflows called in
# release.yml (excluding the release test itself).
#
# Set DRY_RUN=1 to print the discovered matrix without running anything.

nightly_workflow=".github/workflows/e2e_nightly.yml"
nightly_platform_workflow=".github/workflows/e2e_nightly_platform.yml"
regression_workflow=".github/workflows/e2e_regression.yml"
regression_matrix="jobs.regression-test.strategy.matrix"

# Create an associative array to hold platform.name -> test list
declare -A platform_tests

# Append a test to a platform's test list, unless it's already there.
add_test() {
  local platform="$1" test="$2"
  if [[ " ${platform_tests[$platform]-} " != *" $test "* ]]; then
    platform_tests[$platform]+=" $test"
  fi
}

# The nightly workflow fans out over platforms, each of them calling the
# per-platform workflow that holds the test matrix.
echo "Processing $nightly_workflow and $nightly_platform_workflow" >&2

mapfile -t nightly_jobs < <(
  yq -r ".jobs[] | select(.uses == \"./$nightly_platform_workflow\") | [.with.platform-name, .with.debug-set-test-name] | @tsv" "$nightly_workflow"
)
mapfile -t nightly_tests < <(yq ".jobs.test_matrix.strategy.matrix.test-name[]" "$nightly_platform_workflow")

for job in "${nightly_jobs[@]}"; do
  IFS=$'\t' read -r platform debug_set_test <<<"$job"
  for test in "${nightly_tests[@]}"; do
    # The per-platform workflow excludes the gpu test on non-GPU platforms.
    if [[ $test == "gpu" ]] && [[ $platform != *GPU* ]]; then
      continue
    fi
    add_test "$platform" "$test"
  done
  # The matrix includes one test that exercises the debug package set.
  add_test "$platform" "$debug_set_test"
done

# The regression workflow still carries platforms and tests in a single matrix.
echo "Processing $regression_workflow at path $regression_matrix" >&2

mapfile -t regression_tests < <(yq ".$regression_matrix.test-name[]" "$regression_workflow")
platform_count=$(yq ".$regression_matrix.platform | length" "$regression_workflow")

for ((i = 0; i < platform_count; i++)); do
  name=$(yq ".$regression_matrix.platform[$i].name" "$regression_workflow")
  self_hosted=$(yq ".$regression_matrix.platform[$i].self-hosted" "$regression_workflow")

  for test in "${regression_tests[@]}"; do
    if ! yq -o=json ".$regression_matrix.exclude[]?" "$regression_workflow" |
      jq -e --argjson sh "$self_hosted" --arg t "$test" 'select(."test-name" == $t and .platform."self-hosted" == $sh)' >/dev/null; then
      add_test "$name" "$test"
    fi
  done
done

# Apply includes
include_count=$(yq ".$regression_matrix.include | length" "$regression_workflow" 2>/dev/null || echo 0)
for ((j = 0; j < include_count; j++)); do
  name=$(yq ".$regression_matrix.include[$j].platform.name" "$regression_workflow")
  test=$(yq ".$regression_matrix.include[$j].test-name" "$regression_workflow")
  add_test "$name" "$test"
done

# Output merged results
echo "Discovered the following test matrix:" >&2
for platform in "${!platform_tests[@]}"; do
  echo "$platform:${platform_tests[$platform]}" >&2
done

if [[ ${DRY_RUN:-} == "1" ]]; then
  echo "DRY_RUN is set, not running any tests." >&2
  exit 0
fi

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
