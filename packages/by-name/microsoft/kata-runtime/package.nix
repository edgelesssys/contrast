# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  buildGoModule,
  fetchFromGitHub,
  yq-go,
  git,
  applyPatches,
}:

buildGoModule (finalAttrs: {
  pname = "kata-runtime";
  version = "3.2.0.azl2";

  src = applyPatches {
    src = fetchFromGitHub {
      owner = "microsoft";
      repo = "kata-containers";
      rev = finalAttrs.version;
      hash = "sha256-5dLWoVy2+RVq3ssGW7bYYAr3mQdO/ehJphpdJ435pC0=";
    };

    patches = [
      # This allows denying ReadStream requests without blocking the container on its
      # stdout/stderr, by redacting the streams instead of blocking them.
      # Upstream:
      # * https://github.com/kata-containers/kata-containers/issues/10680
      # * https://github.com/kata-containers/kata-containers/pull/10818
      ./0001-agent-clear-log-pipes-if-denied-by-policy.patch
    ];
  };

  sourceRoot = "${finalAttrs.src.name}/src/runtime";

  vendorHash = null;

  subPackages = [
    "cmd/containerd-shim-kata-v2"
    "cmd/kata-monitor"
    # TODO(malt3): enable kata-runtime
    # It depends on CGO and kvm
    # "cmd/kata-runtime"
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

  env.CGO_ENABLED = 0;
  ldflags = [ "-s" ];

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
})
