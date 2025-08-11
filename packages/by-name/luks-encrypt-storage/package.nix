{ writeShellApplication, fetchurl }:
writeShellApplication {
  name = "luks-encrypt-storage";
  program = fetchurl {
    url = "https://github.com/confidential-containers/guest-components/blob/58fbc05c6c3ef48a6232065647b7b92f7965890a/confidential-data-hub/hub/src/storage/scripts/luks-encrypt-storage";
    sha256 = "";
  };
}
