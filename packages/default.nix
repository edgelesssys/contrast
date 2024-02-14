{ lib, pkgs }:


let
  pkgs' = pkgs // self;
  callPackages = lib.callPackagesWith pkgs';
  self = lib.by-name pkgs' ./by-name // {
    containers = callPackages ./containers.nix { pkgs = pkgs'; };
    scripts = callPackages ./scripts.nix { pkgs = pkgs'; };
    genpolicy-msft = pkgs.pkgsStatic.callPackage ./by-name/genpolicy-msft/package.nix { };
    genpolicy-kata = pkgs.pkgsStatic.callPackage ./by-name/genpolicy-kata/package.nix { };
  };
in
self
