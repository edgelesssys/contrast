{ lib, pkgs }:


let
  pkgs' = pkgs // self;
  callPackages = lib.callPackagesWith pkgs';
  self = lib.by-name pkgs' ./by-name // {
    containers = callPackages ./containers.nix { pkgs = pkgs'; };
    scripts = callPackages ./scripts.nix { pkgs = pkgs'; };
  };
in
self
