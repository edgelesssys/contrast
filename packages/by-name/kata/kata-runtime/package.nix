# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildGoModule,
  fetchFromGitHub,
  yq-go,
  git,
  applyPatches,
}:

buildGoModule rec {
  pname = "kata-runtime";
  version = "3.7.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-Ir+/ZZJHm6E+044wczU3UvL+Py9Wprgw2QKJaYyDrKU=";
    };

    patches = [
      ./0001-govmm-Directly-pass-the-firwmare-using-bios-with-SNP.patch
      ./0002-emulate-CPU-model-that-most-closely-matches-the-host.patch
      # This patch makes the v2 shim set the host-data field for SNP and makes
      # kata-agent verify it against the policy.
      # It was adapted from https://github.com/kata-containers/kata-containers/pull/8469,
      # with the following modifications:
      # - Rebase on 3.7.0 picked up regorus, guest-pull capabilities, SNP certificates and TDX fixes.
      # - The TDX parameters for QEMU needed to be converted to a JSON object.
      # - The encoding of MRCONFIGID needed to be switched from hex to base64.
      # - Don't allow skipping the policy checks.
      # - Always use sha256 - even for TDX.
      # This patch is not going to be accepted upstream. The declared path
      # forward is the initdata proposal, https://github.com/kata-containers/kata-containers/issues/9468,
      # which extends the hostdata to arbitrary config beyond the policy and
      # delegates hash verification to the AA. Until that effort lands, we're
      # sticking with the policy verification from AKS CoCo.
      ./0003-runtime-agent-verify-the-agent-policy-hash.patch
      # This patches fixes compatibility issues with QEMU 9.1.
      # https://github.com/kata-containers/kata-containers/pull/10113
      ./0004-qemu-don-t-emit-scsi-parameter.patch
    ];
  };

  sourceRoot = "${src.name}/src/runtime";

  vendorHash = null;

  subPackages = [
    "cmd/containerd-shim-kata-v2"
    "cmd/kata-monitor"
    "cmd/kata-runtime"
  ];

  preBuild = ''
    substituteInPlace Makefile \
      --replace-fail 'include golang.mk' ""
    for f in $(find . -name '*.in' -type f); do
      make ''${f%.in}
    done
  '';

  nativeBuildInputs = [
    yq-go
    git
  ];

  ldflags = [ "-s" ];

  # Hack to skip all kata-runtime tests, which require a Git repo.
  preCheck = ''
    rm cmd/kata-runtime/*_test.go
  '';

  checkFlags =
    let
      # Skip tests that require a working hypervisor
      skippedTests = [
        "TestArchRequiredKernelModules"
        "TestCheckCLIFunctionFail"
        "TestEnvCLIFunction(|Fail)"
        "TestEnvGetAgentInfo"
        "TestEnvGetEnvInfo(|SetsCPUType|NoHypervisorVersion|AgentError|NoOSRelease|NoProcCPUInfo|NoProcVersion)"
        "TestEnvGetRuntimeInfo"
        "TestEnvHandleSettings(|InvalidParams)"
        "TestGetHypervisorInfo"
        "TestGetHypervisorInfoSocket"
        "TestSetCPUtype"
      ];
    in
    [ "-skip=^${builtins.concatStringsSep "$|^" skippedTests}$" ];

  meta.mainProgram = "containerd-shim-kata-v2";
}
