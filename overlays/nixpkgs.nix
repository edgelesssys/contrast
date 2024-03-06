final: prev: {
  # Drop when the update reaches the unstable channel:
  # https://nixpk.gs/pr-tracker.html?pr=293563
  go = prev.go.overrideAttrs (finalAttrs: _prevAttrs: {
    version = "1.21.8";
    src = final.fetchurl {
      url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
      hash = "sha256-3IBs91qH4UFLW0w9y53T6cyY9M/M7EK3r2F9WmWKPEM=";
    };
  });
}
