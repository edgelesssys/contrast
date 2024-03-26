_final: _prev: {
  # Use when a version of Go is needed that is not available in the nixpkgs yet.
  # go = prev.go.overrideAttrs (finalAttrs: _prevAttrs: {
  #   version = "";
  #   src = final.fetchurl {
  #     url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
  #     hash = "";
  #   };
  # });
}
