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
  version = "3.6.0";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "kata-containers";
      repo = "kata-containers";
      rev = version;
      hash = "sha256-Setg6qmkUVn57BQ3wqqNpzmfXeYhJJt9Q4AVFbGrCug=";
    };

    patches = [
      ./0001-govmm-pass-bios-instead-of-if-pflash.patch
      ./0002-emulate-CPU-model-that-most-closely-matches-the-host.patch
      # This patch makes the v2 shim set the host-data field for SNP and makes
      # kata-agent verify it against the policy.
      # Note that these patches are incomplete/insecure.
      ./0003-runtime-agent-verify-the-agent-policy-hash.patch
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
