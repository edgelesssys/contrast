{ inputs }:
final: prev: {
  fenix = inputs.fenix.packages.${final.system};
  craneLib = inputs.crane.mkLib prev.pkgs;
}
