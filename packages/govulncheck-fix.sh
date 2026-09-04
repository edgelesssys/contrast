#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

exec 3>&1 1>&2

readonly overlay="overlays/nixpkgs.nix"
readonly goAttr="legacyPackages.x86_64-linux.base.nixpkgs.go"

readonly sep=$'\x1f'

readonly exitNeedsHuman=4

dryRun=false
if [[ ${1:-} == "--dry-run" ]]; then
  dryRun=true
elif [[ $# -gt 0 ]]; then
  echo "usage: govulncheck-fix [--dry-run]"
  exit 2
fi

workdir=$(mktemp -d)
trap 'rm -rf "${workdir}"' EXIT

tags="${GOVULNCHECK_TAGS}"

# Skips are (osvs, module, current, reason) and actions are (kind, dir, module, current, fixed, osvs).
skip() { printf '%s\n' "$1${sep}$2${sep}$3${sep}$4" >>"${workdir}/skips"; }

relative() {
  if [[ $1 == "${PWD}" ]]; then
    printf '.'
  else
    printf '%s' "${1#"${PWD}"/}"
  fi
}

: >"${workdir}/findings.jsonl"
while IFS= read -r dir; do
  echo "  govulncheck ${dir}"
  CGO_ENABLED=0 govulncheck -C "${dir}" -tags "${tags}" -format json ./... |
    jq -c --arg dir "${dir}" 'select(.finding) | .finding + {dir: $dir}' >>"${workdir}/findings.jsonl"
done < <(go list -f '{{.Dir}}' -m)

if [[ ! -s ${workdir}/findings.jsonl ]]; then
  echo "No vulnerabilities found."
  exit 0
fi

# A single OSV yields module-level, package-level and symbol-level.
# Only the symbol-level one (trace[0].function set) means we call the code path.
jq -s -r '
  [ .[] | select(.trace[0].function != null) ]
  | group_by([ .dir, .trace[0].module ])
  | map([
      .[0].dir,
      .[0].trace[0].module,
      .[0].trace[0].version,
      ([ .[] | .fixed_version // empty ] | unique | join(" ")),
      ([ .[] | select(.fixed_version != null) | .osv ] | unique | join(", ")),
      ([ .[] | select(.fixed_version == null) | .osv ] | unique | join(", "))
    ])
  | .[] | join("\u001f")
' "${workdir}/findings.jsonl" >"${workdir}/targets"

: >"${workdir}/actions"
: >"${workdir}/skips"
: >"${workdir}/stdlib"

while IFS="${sep}" read -r dir module current fixedVersions osvs unfixed; do
  if [[ -n ${unfixed} ]]; then
    skip "${unfixed}" "${module}" "${current}" "no fixed version released yet"
  fi

  if [[ -z ${fixedVersions} ]]; then
    continue
  fi

  fixed=$(printf '%s' "${fixedVersions}" | tr ' ' '\n' | sort -V | tail -n1)

  if [[ ${module} == "stdlib" || ${module} == "toolchain" ]]; then
    printf '%s\n' "${osvs}${sep}${current}${sep}${fixed}" >>"${workdir}/stdlib"
    continue
  fi

  # v0 and v1 share the import path, only v2+ moves the module to a /vN path.
  fixedMajor="${fixed%%.*}"
  if [[ ${fixedMajor} != "v0" && ${fixedMajor} != "v1" && ${fixedMajor} != "${current%%.*}" ]]; then
    skip "${osvs}" "${module}" "${current}" "needs a major version bump (${current} -> ${fixed})"
    continue
  fi

  printf '%s\n' "gomod${sep}${dir}${sep}${module}${sep}${current}${sep}${fixed}${sep}${osvs}" \
    >>"${workdir}/actions"
done <"${workdir}/targets"

if [[ -s ${workdir}/stdlib ]]; then
  stdlibCurrent=$(cut -d"${sep}" -f2 "${workdir}/stdlib" | sort -V | tail -n1 | sed -E 's/^(go|v)//')
  stdlibFixed=$(cut -d"${sep}" -f3 "${workdir}/stdlib" | sort -V | tail -n1 | sed -E 's/^(go|v)//')
  stdlibOsvs=$(cut -d"${sep}" -f1 "${workdir}/stdlib" | tr ',' '\n' | tr -d ' ' |
    grep -v '^$' | sort -u | paste -sd',' - | sed 's/,/, /g')

  runningMinor=$(go version | sed -E 's/.*go([0-9]+\.[0-9]+).*/\1/')
  goPin="go_${runningMinor//./_}"

  if [[ ${stdlibFixed%.*} != "${runningMinor}" ]]; then
    skip "${stdlibOsvs}" "stdlib" "${stdlibCurrent}" "needs go ${stdlibFixed}, pinning go_${stdlibFixed%.*} would not override the default \`go\`"
  elif grep -qE '^  go_1_[0-9]+ = prev' "${overlay}" &&
    ! grep -q "${goPin} = prev.${goPin}.overrideAttrs" "${overlay}"; then
    skip "${stdlibOsvs}" "stdlib" "${stdlibCurrent}" \
      "${overlay} pins a go minor other than ${runningMinor}, remove that pin first"
  else
    printf '%s\n' "stdlib${sep}${sep}stdlib${sep}${stdlibCurrent}${sep}${stdlibFixed}${sep}${stdlibOsvs}" \
      >>"${workdir}/actions"
  fi
fi

: >"${workdir}/applied"

writeReport() {
  if [[ -s ${workdir}/applied ]]; then
    if [[ ${dryRun} == true ]]; then
      printf '### Would fix automatically\n\n'
    else
      printf '### Fixed automatically\n\n'
    fi
    printf '| Advisory | Module | From | To | Where |\n|---|---|---|---|---|\n'
    while IFS="${sep}" read -r kind dir module current fixed osvs; do
      if [[ ${kind} == stdlib ]]; then
        printf '| %s | go (toolchain) | %s | %s | %s |\n' \
          "${osvs}" "${current}" "${fixed}" "${overlay}"
      else
        printf '| %s | %s | %s | %s | %s |\n' \
          "${osvs}" "${module}" "${current}" "${fixed}" "$(relative "${dir}")"
      fi
    done <"${workdir}/applied"
    printf '\n'
  fi
  if [[ -s ${workdir}/skips ]]; then
    printf '### Needs a human\n\n'
    printf '| Advisory | Module | Version | Reason |\n|---|---|---|---|\n'
    sort -u "${workdir}/skips" |
      while IFS="${sep}" read -r osvs module current reason; do
        printf '| %s | %s | %s | %s |\n' "${osvs}" "${module}" "${current}" "${reason}"
      done
    printf '\n'
  fi
} >&3

if [[ ! -s ${workdir}/actions ]]; then
  writeReport
  echo "Nothing to fix automatically."
  if [[ -s ${workdir}/skips ]]; then
    exit "${exitNeedsHuman}"
  fi
  exit 0
fi

if [[ ${dryRun} == true ]]; then
  cp "${workdir}/actions" "${workdir}/applied"
  writeReport
  exit 0
fi

{
  awk -F"${sep}" '$1 == "stdlib"' "${workdir}/actions"
  awk -F"${sep}" '$1 != "stdlib"' "${workdir}/actions"
} >"${workdir}/ordered"

while IFS="${sep}" read -r kind dir module current fixed osvs; do
  if [[ ${kind} == stdlib ]]; then
    echo "Pinning go to ${fixed} in ${overlay}"
    if ! grep -q "${goPin} = prev.${goPin}.overrideAttrs" "${overlay}"; then
      awk -v attr="${goPin}" '
        { print }
        /^  # }\);$/ && !done {
          print ""
          print "  # Remove this once nixpkgs has caught up."
          print "  " attr " = prev." attr ".overrideAttrs ("
          print "    finalAttrs: _prevAttrs: {"
          print "      version = \"0\";"
          print "      src = final.fetchurl {"
          print "        url = \"https://go.dev/dl/go${finalAttrs.version}.src.tar.gz\";"
          print "        hash = \"sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\";"
          print "      };"
          print "    }"
          print "  );"
          done = 1
        }
      ' "${overlay}" >"${workdir}/overlay.nix"
      mv "${workdir}/overlay.nix" "${overlay}"
      grep -q "${goPin} = prev.${goPin}.overrideAttrs" "${overlay}"
    fi
    nix-update --flake --version="${fixed}" --override-filename="${overlay}" "${goAttr}"
  else
    echo "go get ${module}@${fixed} in ${dir}"
    if ! go get -C "${dir}" "${module}@${fixed}"; then
      skip "${osvs}" "${module}" "${current}" \
        "\`go get ${module}@${fixed}\` failed, may need a toolchain bump first"
      continue
    fi
  fi
  printf '%s\n' "${kind}${sep}${dir}${sep}${module}${sep}${current}${sep}${fixed}${sep}${osvs}" \
    >>"${workdir}/applied"
done <"${workdir}/ordered"

if [[ ! -s ${workdir}/applied ]]; then
  writeReport
  echo "Could not apply any fix."
  exit "${exitNeedsHuman}"
fi

echo "Running codegen"
generate

writeReport

if [[ -s ${workdir}/skips ]]; then
  echo "Applied mechanical fixes, but some findings need a human."
  exit "${exitNeedsHuman}"
fi
