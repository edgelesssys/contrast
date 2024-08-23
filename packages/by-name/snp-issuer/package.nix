{ buildGoModule }:

buildGoModule {
  pname = "snp-issuer";
  version = "0.0.1";

  src = ./../../../tools/snp-issuer;

  vendorHash = "sha256-LOFI+0fj78IeP+lZZW480CR/ULLXFfY6Ra0bxg3yt8U=";
  proxyVendor = true;

  CGO_ENABLED = "0";

  ldflags = [ "-s" ];

  meta.mainProgram = "snp-issuer";
}
