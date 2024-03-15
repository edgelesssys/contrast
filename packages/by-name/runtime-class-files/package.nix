{ fetchurl
, stdenvNoCC
, igvmmeasure
}:
let
  rootfs = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers.img";
    hash = "sha256-EdFywKAU+xD0BXmmfbjV4cB6Gqbq9R9AnMWoZFCM3A0=";
  };
  igvm = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/kata-containers-igvm.img";
    hash = "sha256-E9Ttx6f9QYwKlQonO/fl1bF2MNBoU4XG3/HHvt9Zv30=";
  };
  cloud-hypervisor-bin = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/cloud-hypervisor-cvm";
    hash = "sha256-coTHzd5/QLjlPQfrp9d2TJTIXKNuANTN7aNmpa8PRXo=";
  };
  containerd-shim-contrast-cc-v2 = fetchurl {
    url = "https://cdn.confidential.cloud/contrast/node-components/2024-03-13/containerd-shim-kata-cc-v2";
    hash = "sha256-yhk3ZearqQVz1X1p67OFPCSHbF0P66E7KknpO/JGzZg=";
  };
in
stdenvNoCC.mkDerivation {
  name = "runtime-class-files";
  version = "2024-03-13";

  dontUnpack = true;

  buildInputs = [ igvmmeasure ];

  buildPhase = ''
    mkdir -p $out
    igvmmeasure -b ${igvm} | dd conv=lcase > $out/launch-digest.hex
  '';

  passthru = {
    inherit rootfs igvm cloud-hypervisor-bin containerd-shim-contrast-cc-v2;
  };
}
